package server

import (
	"errors"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/strutil"
)

// recordMail appends a display-only record of one send or read to the caller's
// chat transcript (TS-04.R108). A caller without a live chat transcript gets
// none. A failed append leaves the tool result unchanged and logs no content.
func (s *Server) recordMail(agentID string, mail runtime.MailNotice) {
	err := s.registry.AppendMailNotice(agentID, mailRecordNotice(mail))
	if err == nil || errors.Is(err, runtime.ErrNoHandle) || errors.Is(err, runtime.ErrNotImplemented) {
		return
	}
	s.log.Warn("mail record not appended", "agent", agentID, "direction", mail.Direction)
}

// mailRecordNotice keeps the collapsed line short; the detail carries content.
func mailRecordNotice(mail runtime.MailNotice) runtime.NoticeData {
	title := "Mail read"
	if mail.Direction == "sent" {
		names := make([]string, 0, len(mail.Messages))
		for _, m := range mail.Messages {
			names = append(names, strutil.FirstNonEmpty(m.ToName, m.ToAddress))
		}
		title = "Mail to " + strings.Join(names, ", ")
		if mail.Error != nil {
			title = "Mail not sent"
		}
	}
	return runtime.MailNoticeData(title, "", mail)
}
