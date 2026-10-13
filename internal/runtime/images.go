package runtime

import (
	"encoding/base64"
	"github.com/AsaphNoam/Chuck/internal/attachment"
	"io"
	"net/url"
	"strings"
)

func (c *ChatRuntime) persistImage(as *agentState, b acpContentBlock, toolID string) AssistantImageData {
	d := AssistantImageData{ToolCallID: toolID}
	if b.Data == "" {
		if u, err := url.Parse(b.URI); err == nil && u.Scheme == "file" && (u.Host == "" || u.Host == "localhost") {
			d.Path = u.Path
			return d
		}
		d.Error = "image bytes unavailable"
		return d
	}
	if len(b.Data) > base64.StdEncoding.EncodedLen(10*1024*1024) {
		d.Error = "image exceeds 10 MiB limit"
		return d
	}
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(b.Data))
	c.mu.Lock()
	home := c.transcriptHome
	c.mu.Unlock()
	ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}[b.MIMEType]
	if ext == "" {
		d.Error = "unsupported image format"
		return d
	}
	a, err := attachment.New(home).Put(as.agentID, "generated-image"+ext, io.LimitReader(reader, 10*1024*1024+1))
	if err != nil {
		d.Error = "could not retain image"
		return d
	}
	if a.Kind != "image" || a.MIMEType != b.MIMEType {
		_ = attachment.New(home).Remove(as.agentID, a.ID)
		d.Error = "image bytes do not match image format"
		return d
	}
	d.Attachment = &a
	return d
}
