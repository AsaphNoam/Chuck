package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

const attachmentRequestLimit = 26 << 20
const attachmentMessageLimit = 64 << 10

var errAttachmentBusy = errors.New("another message is being prepared; retry shortly")

func isMultipartMessageRoute(r *http.Request) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "multipart/form-data" || r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	return len(parts) == 4 && parts[0] == "api" && parts[1] == "sessions" && parts[2] != "" && (parts[3] == "prompt" || parts[3] == "steer")
}

func (s *Server) knownAttachmentSession(id string) *runtime.APIError {
	a, err := s.stateStore.ReadAgent(id)
	if errors.Is(err, state.ErrNotFound) {
		return apiError(runtime.CodeNotFound, "no such conversation")
	}
	if err != nil {
		return apiError(runtime.CodeInternal, "cannot read conversation")
	}
	if a.Interface != "chat" {
		return apiError(runtime.CodeConflict, "images are only available for chat conversations")
	}
	if _, err = s.stateStore.ReadSession(id); errors.Is(err, state.ErrNotFound) {
		return apiError(runtime.CodeNotFound, "no such conversation")
	} else if err != nil {
		return apiError(runtime.CodeInternal, "cannot read conversation")
	}
	return nil
}

func serveAttachmentBytes(w http.ResponseWriter, r *http.Request, f *os.File, d attachment.Descriptor) {
	w.Header().Set("Content-Type", d.MIMEType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	disposition := "attachment"
	if d.Kind == attachment.KindImage {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": d.Name}))
	// No range/If-Modified cache behavior: validated immutable bytes are private.
	w.Header().Set("Content-Length", fmt.Sprint(d.Size))
	_, _ = io.CopyN(w, f, d.Size)
}

func (s *Server) handleAttachmentRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if ae := s.knownAttachmentSession(id); ae != nil {
		writeAPIError(w, ae)
		return
	}
	f, d, err := attachment.New(s.configStore.Home()).Open(id, r.PathValue("attachment_id"))
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "uploaded attachment is unavailable"))
		return
	}
	defer f.Close()
	serveAttachmentBytes(w, r, f, d)
}

func (s *Server) handleImageRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if ae := s.knownAttachmentSession(id); ae != nil {
		writeAPIError(w, ae)
		return
	}
	snap, err := s.stateStore.ReadSession(id)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, "cannot read conversation"))
		return
	}
	path, display, ae := resolveFileReadPath(snap.Cwd, r.URL.Query().Get("path"))
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	noFollow := remoteNoFollowFileRead(r.Context())
	workspaceRoot := ""
	if noFollow {
		root, ae := resolveWorkspaceRoot(snap.Cwd)
		if ae != nil {
			writeAPIError(w, ae)
			return
		}
		workspaceRoot = root
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			writeRemoteError(w, http.StatusNotFound, codeRemoteFileNotTracked, "that image is outside this workspace")
			return
		}
		current := root
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			st, err := os.Lstat(current)
			if err != nil || st.Mode()&os.ModeSymlink != 0 {
				writeRemoteError(w, http.StatusNotFound, codeRemoteFileNotTracked, "that image is unavailable or uses a symbolic link")
				return
			}
		}
	}
	// Classify before open to reject known special files; nonblocking open and
	// descriptor classification also close a replacement race with FIFO/devices.
	stat := os.Stat
	if noFollow {
		stat = os.Lstat
	}
	st, err := stat(path)
	if err != nil || !st.Mode().IsRegular() {
		writeAPIError(w, apiError(runtime.CodeNotAFile, "that image is not an available regular file"))
		return
	}
	flags := syscall.O_RDONLY | syscall.O_NONBLOCK | syscall.O_CLOEXEC
	if noFollow {
		flags |= syscall.O_NOFOLLOW
	}
	fd, err := syscall.Open(path, flags, 0)
	if noFollow {
		if err == nil {
			syscall.Close(fd)
		}
		fd, err = openWorkspaceImage(workspaceRoot, path)
	}
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeFileUnreadable, "that image cannot be read"))
		return
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(st, opened) || opened.Size() > attachment.MaxFileBytes {
		writeAPIError(w, apiError(runtime.CodeValidation, "image is unavailable or exceeds 10 MiB"))
		return
	}
	d, err := attachment.Inspect(f, filepath.Base(display))
	if err != nil || d.Kind != attachment.KindImage {
		writeAPIError(w, apiError(runtime.CodeValidation, "image format or contents are unsupported"))
		return
	}
	d.Size = opened.Size()
	d.Name = filepath.Base(display)
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		writeAPIError(w, apiError(runtime.CodeFileUnreadable, "that image cannot be read"))
		return
	}
	serveAttachmentBytes(w, r, f, d)
}

// Each relative directory is opened through the previously owned descriptor,
// so replacing an intermediate path with a symlink cannot escape the phone's
// tracked workspace boundary between inspection and open.
func openWorkspaceImage(root, path string) (int, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return -1, fmt.Errorf("image outside workspace")
	}
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for i, part := range parts {
		flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC
		if i < len(parts)-1 {
			flags |= unix.O_DIRECTORY
		}
		next, err := unix.Openat(fd, part, flags, 0)
		unix.Close(fd)
		if err != nil {
			return -1, err
		}
		fd = next
	}
	return fd, nil
}

type messageRequest struct {
	RequestID string `json:"request_id"`
	Text      string `json:"text"`
	Files     []struct {
		Part string `json:"part"`
		Name string `json:"name"`
	} `json:"files"`
	AttachmentIDs []string `json:"attachment_ids"`
}

// Admission covers parsing and acceptance, limiting both temporary storage and
// concurrent retries before any request bytes are consumed.
func (s *Server) claimAttachmentUpload(id string) bool {
	s.attachmentMu.Lock()
	defer s.attachmentMu.Unlock()
	if s.attachmentBusy[id] || len(s.attachmentBusy) >= 4 {
		return false
	}
	if s.attachmentBusy == nil {
		s.attachmentBusy = map[string]bool{}
	}
	s.attachmentBusy[id] = true
	return true
}
func (s *Server) releaseAttachmentUpload(id string) {
	s.attachmentMu.Lock()
	delete(s.attachmentBusy, id)
	s.attachmentMu.Unlock()
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func decodeMessageJSON(r io.Reader, out *messageRequest) error {
	data, err := io.ReadAll(io.LimitReader(r, remoteBodyLimit+1))
	if err != nil {
		return err
	}
	if len(data) > remoteBodyLimit || !utf8.Valid(data) {
		return fmt.Errorf("invalid or oversized JSON metadata")
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("unexpected message data")
	}
	return nil
}

func attachmentAdmissionError(err error) *runtime.APIError {
	var bodyLimit *http.MaxBytesError
	var pathError *os.PathError
	switch {
	case errors.Is(err, errAttachmentBusy):
		return apiError(runtime.CodeConflict, err.Error())
	case errors.As(err, &bodyLimit), errors.Is(err, attachment.ErrFileTooLarge), errors.Is(err, attachment.ErrTotalTooLarge), errors.Is(err, attachment.ErrTooManyFiles):
		return apiError(runtime.CodeAttachmentLimit, "Choose up to 10 files, at most 10 MiB each and 25 MiB total")
	case errors.Is(err, attachment.ErrInvalidImage):
		return apiError(runtime.CodeAttachmentType, "Image contents do not match a supported PNG, JPEG, WebP or GIF format")
	case errors.As(err, &pathError):
		return apiError(runtime.CodeAttachmentStorage, "Could not retain attachments; check available local storage and retry")
	default:
		return apiError(runtime.CodeValidation, err.Error())
	}
}

// parseMessage streams uploads directly into bounded owned storage. Until a
// receipt accepts them, every created blob is owned by this request's rollback.
func (s *Server) parseMessage(w http.ResponseWriter, r *http.Request, claimed *bool) (runtime.Message, []string, error) {
	id := r.PathValue("id")
	store := attachment.New(s.configStore.Home())
	var req messageRequest
	msg := runtime.Message{}
	created := []string{}
	fail := func(err error) (runtime.Message, []string, error) { return msg, created, err }
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil && r.Header.Get("Content-Type") != "" {
		return fail(fmt.Errorf("invalid content type"))
	}
	if media == "multipart/form-data" {
		r.Body = http.MaxBytesReader(w, r.Body, attachmentRequestLimit)
		mr, err := r.MultipartReader()
		if err != nil {
			return fail(fmt.Errorf("invalid multipart body"))
		}
		part, err := mr.NextPart()
		if err != nil || part.FormName() != "message" || part.FileName() != "" {
			return fail(fmt.Errorf("first part must be message metadata"))
		}
		data, err := io.ReadAll(io.LimitReader(part, attachmentMessageLimit+1))
		if err != nil || len(data) > attachmentMessageLimit {
			return fail(fmt.Errorf("message metadata exceeds 64 KiB"))
		}
		if err = decodeMessageJSON(strings.NewReader(string(data)), &req); err != nil {
			return fail(fmt.Errorf("invalid message metadata"))
		}
		if len(req.Files) == 0 || len(req.Files) > attachment.MaxFiles || len(req.AttachmentIDs) > 0 {
			return fail(attachment.ErrTooManyFiles)
		}
		if !validRequestID(req.RequestID) || !utf8.ValidString(req.Text) {
			return fail(fmt.Errorf("invalid request id or text"))
		}
		seen := map[string]bool{}
		var total int64
		for _, decl := range req.Files {
			if !validRequestID(decl.Part) || decl.Part == "message" || seen[decl.Part] {
				return fail(fmt.Errorf("duplicate or invalid file part"))
			}
			seen[decl.Part] = true
			part, err = mr.NextPart()
			if err != nil || part.FormName() != decl.Part {
				return fail(fmt.Errorf("missing or out-of-order file part"))
			}
			d, err := store.Put(id, decl.Name, io.LimitReader(part, attachment.MaxTotalBytes-total+1))
			if err != nil {
				return fail(err)
			}
			created = append(created, d.ID)
			total += d.Size
			if total > attachment.MaxTotalBytes {
				return fail(attachment.ErrTotalTooLarge)
			}
			msg.Attachments = append(msg.Attachments, d)
		}
		if _, err = mr.NextPart(); err != io.EOF {
			return fail(fmt.Errorf("undeclared or incomplete file part"))
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, remoteBodyLimit)
		if err := decodeMessageJSON(r.Body, &req); err != nil {
			return fail(fmt.Errorf("invalid JSON body"))
		}
		if len(req.Files) > 0 {
			return fail(fmt.Errorf("files require multipart data"))
		}
		if len(req.AttachmentIDs) > attachment.MaxFiles {
			return fail(attachment.ErrTooManyFiles)
		}
		if len(req.AttachmentIDs) > 0 && !*claimed {
			if !s.claimAttachmentUpload(id) {
				return fail(errAttachmentBusy)
			}
			*claimed = true
		}
		seen := map[string]bool{}
		var total int64
		for _, assetID := range req.AttachmentIDs {
			if seen[assetID] {
				return fail(fmt.Errorf("duplicate attachment id"))
			}
			seen[assetID] = true
			f, d, err := store.Open(id, assetID)
			if err != nil {
				return fail(fmt.Errorf("uploaded attachment is unavailable"))
			}
			f.Close()
			total += d.Size
			if total > attachment.MaxTotalBytes {
				return fail(attachment.ErrTotalTooLarge)
			}
			msg.Attachments = append(msg.Attachments, d)
		}
		if len(msg.Attachments) > 0 && !validRequestID(req.RequestID) {
			return fail(fmt.Errorf("attachment request id is required"))
		}
	}
	msg.Text = req.Text
	msg.RequestID = req.RequestID
	if msg.RequestID != "" && !validRequestID(msg.RequestID) {
		return fail(fmt.Errorf("invalid request id"))
	}
	if len(msg.Attachments) > 0 {
		encoded, err := json.Marshal(msg)
		if err != nil || len(encoded) > 96<<10 || len(msg.Text) > attachmentMessageLimit {
			return fail(attachment.ErrTotalTooLarge)
		}
	}
	return msg, created, nil
}

func messagePayloadHash(action string, msg runtime.Message) string {
	// Opaque ids differ on a multipart retry; hashes and ordered metadata define
	// the payload. The action prevents reusing Send's receipt for Steer.
	type item struct {
		Name, MIME, Hash, Kind string
		Size                   int64
	}
	files := []item{}
	for _, d := range msg.Attachments {
		files = append(files, item{d.Name, d.MIMEType, d.SHA256, d.Kind, d.Size})
	}
	data, _ := json.Marshal(struct {
		Action, Text string
		Files        []item
	}{action, msg.Text, files})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Server) handleMessageAction(w http.ResponseWriter, r *http.Request, action string) {
	id := r.PathValue("id")
	agent, err := s.stateStore.ReadAgent(id)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent"))
		} else {
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot read conversation"))
		}
		return
	}
	if agent.Interface != "chat" && isMultipartMessageRoute(r) {
		writeAPIError(w, apiError(runtime.CodeConflict, "attachments and prompts require a chat agent"))
		return
	}
	claimed := false
	if isMultipartMessageRoute(r) && !s.claimAttachmentUpload(id) {
		writeAPIError(w, apiError(runtime.CodeConflict, "another message is being prepared; retry shortly"))
		return
	}
	claimed = isMultipartMessageRoute(r)
	defer func() {
		if claimed {
			s.releaseAttachmentUpload(id)
		}
	}()
	msg, created, err := s.parseMessage(w, r, &claimed)
	owned := false
	defer func() {
		if !owned {
			for _, assetID := range created {
				_ = attachment.New(s.configStore.Home()).Remove(id, assetID)
			}
		}
	}()
	if err != nil {
		writeAPIError(w, attachmentAdmissionError(err))
		return
	}
	if action == "prompt" && strings.TrimSpace(msg.Text) == "" && len(msg.Attachments) == 0 {
		writeAPIError(w, apiError(runtime.CodeValidation, "text or attachments are required"))
		return
	}
	// Legacy requests receive a recovery identity too, without promising that
	// a later id-less request is a replay of this command.
	promoting := action == "steer" && strings.TrimSpace(msg.Text) == "" && len(msg.Attachments) == 0
	receiptMessage := msg
	if promoting {
		if heldMessage, _, heldErr := s.registry.HeldMessage(id); heldErr == nil {
			receiptMessage = heldMessage
		}
	}
	if receiptMessage.RequestID == "" && (receiptMessage.Text != "" || len(receiptMessage.Attachments) > 0) {
		receiptMessage.RequestID = mintHookToken()
	}
	if promoting {
		msg.RequestID = receiptMessage.RequestID
		msg.HeldID = receiptMessage.HeldID
	} else {
		msg = receiptMessage
	}
	receiptBearing := receiptMessage.RequestID != "" && (receiptMessage.Text != "" || len(receiptMessage.Attachments) > 0)
	if receiptBearing && s.attachmentStartupErr != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, "attachment recovery is unavailable; restart after repairing storage"))
		return
	}
	hash := messagePayloadHash(action, receiptMessage)
	if receiptBearing {
		old, err := s.stateStore.AttachmentReceipt(id, msg.RequestID)
		promoteReceipt := err == nil && promoting && old.Delivery == "held"
		if err == nil {
			// Promoting an existing hold advances its original receipt through
			// the runtime; it is not a second submission of that Send.
			if !promoteReceipt && old.PayloadHash != hash {
				writeAPIError(w, apiError(runtime.CodeConflict, "request id already belongs to different content"))
				return
			}
			if !promoteReceipt {
				s.writeAttachmentAcceptance(w, old)
				return
			}
		}
		if err != nil && !errors.Is(err, state.ErrNotFound) {
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot read message receipt"))
			return
		}
		if !promoteReceipt {
			data, _ := json.Marshal(receiptMessage)
			if err := s.stateStore.CreateAttachmentReceipt(state.AttachmentReceipt{AgentID: id, RequestID: msg.RequestID, PayloadHash: hash, MessageJSON: string(data), Action: action}); err != nil {
				writeAPIError(w, apiError(runtime.CodeInternal, "cannot retain message receipt"))
				return
			}
			owned = true
		}
		// The receipt remains authoritative if removing the provisional marker
		// fails; startup retains its referenced bytes before pruning anything.
		if err := attachment.New(s.configStore.Home()).Retain(id, created); err != nil {
			s.log.Error("retain accepted attachment markers", "agent", id, "err", err)
		}
	}
	held := false
	outcome := runtime.SteerOutcome("")
	if action == "steer" {
		if strings.TrimSpace(msg.Text) == "" && len(msg.Attachments) == 0 {
			msg.Text = ""
		}
		outcome, err = s.registry.SteerMessage(r.Context(), id, msg)
	} else {
		held, err = s.registry.SendMessageOrHold(r.Context(), id, msg)
		if errors.Is(err, runtime.ErrNoHandle) {
			woken, ae := s.wakeAgent(r.Context(), id)
			if ae != nil {
				err = ae
			} else if woken {
				held, err = s.registry.SendMessageOrHold(r.Context(), id, msg)
			}
		}
	}
	if err != nil {
		if receiptBearing {
			// Only a receipt still untouched by runtime delivery can roll back. Once
			// submitted or held, return acceptance so a network retry cannot consume twice.
			rec, readErr := s.stateStore.AttachmentReceipt(id, msg.RequestID)
			if action == "steer" && errors.Is(err, runtime.ErrSteerRefused) {
				writeAPIError(w, apiError(runtime.CodeInternal, steerRefusalText(err)))
				return
			}
			if readErr == nil && rec.Delivery == "failed" {
				// A definite adapter refusal stays retained for recovery but is
				// reported as the refusal, never as a sent message (FS-03.R50).
				if action == "steer" {
					writeAPIError(w, apiError(runtime.CodeInternal, steerRefusalText(err)))
				} else if ae, ok := err.(*runtime.APIError); ok {
					writeAPIError(w, ae)
				} else {
					writeAPIError(w, sessionOpError(err))
				}
				return
			}
			if readErr == nil && errors.Is(err, runtime.ErrMessageAccepted) {
				s.writeAttachmentAcceptance(w, rec)
				return
			}
			if readErr == nil && rec.Delivery != "prepared" {
				s.writeAttachmentAcceptance(w, rec)
				return
			}
			if readErr != nil {
				writeAPIError(w, apiError(runtime.CodeInternal, "cannot verify message acceptance"))
				return
			}
			if removeErr := s.stateStore.RemoveAttachmentReceipt(id, msg.RequestID); removeErr != nil {
				s.writeAttachmentAcceptance(w, rec)
				return
			}
			owned = false
		}
		if errors.Is(err, runtime.ErrAttachmentCapability) {
			writeAPIError(w, apiError(runtime.CodeAttachmentCapability, "This agent does not support native image input; select an image-capable runtime or remove the image"))
			return
		}
		if ae, ok := err.(*runtime.APIError); ok {
			writeAPIError(w, ae)
			return
		}
		if action == "steer" {
			writeAPIError(w, steerError(err))
		} else {
			writeAPIError(w, sessionOpError(err))
		}
		return
	}
	if receiptBearing {
		if outcome != "" {
			_ = s.stateStore.SetAttachmentOutcome(id, msg.RequestID, string(outcome))
		}
		rec, err := s.stateStore.AttachmentReceipt(id, msg.RequestID)
		if err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "message accepted; receipt temporarily unavailable"))
			return
		}
		// Explicit resubmission acknowledges only the recovery that supplied these
		// exact owned copies, leaving a newer independent released record intact.
		if len(created) == 0 {
			if recovered, e := s.stateStore.AttachmentRecovery(id); e == nil {
				var previous runtime.Message
				if json.Unmarshal([]byte(recovered.MessageJSON), &previous) == nil && sameAttachmentIDs(previous, msg) {
					_ = s.stateStore.DismissAttachmentRecovery(id, recovered.RequestID)
				}
			}
		}
		if action == "steer" {
			rec.Action = "steer"
		}
		s.writeAttachmentAcceptance(w, rec)
		return
	}
	if action == "steer" {
		writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "agent_id": id, "outcome": string(outcome)})
		return
	}
	after := int64(0)
	if held {
		_, after, _ = s.registry.Held(id)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "agent_id": id, "delivery": promptDelivery(held), "after_seq": after})
}

func sameAttachmentIDs(a, b runtime.Message) bool {
	if len(a.Attachments) != len(b.Attachments) {
		return false
	}
	for i := range a.Attachments {
		if a.Attachments[i].ID != b.Attachments[i].ID {
			return false
		}
	}
	return true
}

// receiptDelivery reports only a submitted receipt as sent; an uncertain,
// released or failed receipt keeps its own name so no caller reads it as sent.
func receiptDelivery(delivery string) string {
	if delivery == "submitted" {
		return "sent"
	}
	return delivery
}

// steerRefusalText keeps the adapter's own reason without the runtime's
// retained-bytes marker joined to it.
func steerRefusalText(err error) string {
	reason := ""
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range joined.Unwrap() {
			if !errors.Is(e, runtime.ErrMessageAccepted) {
				reason = e.Error()
			}
		}
	}
	if reason == "" && err != nil && !errors.Is(err, runtime.ErrMessageAccepted) {
		reason = err.Error()
	}
	text := "The agent refused this steered message; nothing was delivered."
	if reason != "" {
		text += " " + reason
	}
	return text
}

func (s *Server) writeAttachmentAcceptance(w http.ResponseWriter, rec state.AttachmentReceipt) {
	if rec.Delivery == "prepared" {
		rec.Delivery = "unknown"
	}
	if rec.Delivery == "unknown" {
		err := apiError(runtime.CodeDeliveryUncertain, "This message may already have reached the agent. Inspect history; recover it only with a deliberate new Send.")
		err.Details = map[string]any{"agent_id": rec.AgentID, "request_id": rec.RequestID, "recovery": "/api/sessions/" + rec.AgentID + "/prompt"}
		writeAPIError(w, err)
		return
	}
	if rec.Delivery == "failed" {
		if rec.Action == "steer" {
			writeAPIError(w, apiError(runtime.CodeInternal, steerRefusalText(nil)))
		} else {
			writeAPIError(w, apiError(runtime.CodeConflict, "The message was refused and was not delivered; send it again deliberately."))
		}
		return
	}
	var msg runtime.Message
	_ = json.Unmarshal([]byte(rec.MessageJSON), &msg)
	attachments := append([]attachment.Descriptor{}, msg.Attachments...)
	after := int64(0)
	heldID := ""
	if rec.Delivery == "held" {
		_, after, heldID, _ = s.registry.HeldSnapshot(rec.AgentID)
	}
	body := map[string]any{"accepted": true, "agent_id": rec.AgentID, "request_id": rec.RequestID, "attachments": attachments, "attachment_delivery": rec.Delivery, "delivery": receiptDelivery(rec.Delivery), "after_seq": after, "held_id": heldID}
	if rec.Action == "steer" {
		body["outcome"] = rec.Outcome
	}
	writeJSON(w, http.StatusAccepted, body)
}
