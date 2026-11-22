package state

import "testing"

// FS-01.R36, INV §20: a snapshot with no fork report is unknown, distinct from
// an explicit refusal, for Clone and Edit alike, and never offers the fork.
func TestForkGatesDistinguishNeverReportedFromRefused(t *testing.T) {
	for _, tc := range []struct {
		name, raw       string
		clone, edit     string
		cloneOK, editOK bool
	}{
		{"never reported", `{}`, CloneReasonForkUnknown, EditReasonForkUnknown, false, false},
		{"empty column", ``, CloneReasonForkUnknown, EditReasonForkUnknown, false, false},
		{"malformed", `{"fork":"yes"}`, CloneReasonForkUnknown, EditReasonForkUnknown, false, false},
		{"reported false", `{"fork":false}`, CloneReasonNoFork, EditReasonNoFork, false, false},
		{"reported true", `{"fork":true}`, "", "", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caps := DecodeRuntimeCapabilities(tc.raw)
			if got := CloneAvailability("chat", false, caps, "s1", false); got.Available != tc.cloneOK || got.Reason != tc.clone {
				t.Fatalf("clone = %+v, want %v %q", got, tc.cloneOK, tc.clone)
			}
			a := AgentState{Interface: "chat", State: "idle", RuntimeCapabilities: caps}
			if got := EditAvailability(a, "s1", false); got.Available != tc.editOK || got.Reason != tc.edit {
				t.Fatalf("edit = %+v, want %v %q", got, tc.editOK, tc.edit)
			}
		})
	}
}
