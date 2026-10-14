package state

import (
	"encoding/json"
	"testing"
)

// FS-03.A64/A65: a legacy unknown must survive serialization, while a real
// boolean refusal remains available to both composers.
func TestStoredImageAdvertisementPreservesUnknownAndFalse(t *testing.T) {
	for _, tc := range []struct {
		name, raw    string
		known, image bool
	}{
		{"old snapshot", `{"fork":true}`, false, false},
		{"empty", ``, false, false},
		{"null", `{"image_input":null}`, false, false},
		{"malformed", `{"image_input":"yes"}`, false, false},
		{"refused", `{"image_input":false}`, true, false},
		{"supported", `{"image_input":true}`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caps := DecodeRuntimeCapabilities(tc.raw)
			raw, err := json.Marshal(caps)
			if err != nil {
				t.Fatal(err)
			}
			caps = DecodeRuntimeCapabilities(string(raw))
			projection := AttachmentCapabilitiesFor("chat", caps)
			if got := projection.ImageInput; (got != nil) != tc.known || (got != nil && *got != tc.image) {
				t.Fatalf("image advertisement = %v after %s", got, raw)
			}
			if AttachmentCapabilitiesFor("terminal", caps) != nil {
				t.Fatal("terminal acquired attachment support")
			}
		})
	}
}
