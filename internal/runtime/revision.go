package runtime

// EvConversationRevision records that a committed last-message Edit replaced
// a range of the active conversation (FS-01.R61, TS-02.R69–R70). It is
// metadata: it carries no message text or native ids and never settles quota,
// task, room, mail or turn control. The journal keeps the superseded events'
// bytes; readers hide them through transcript's revision fold.
const EvConversationRevision = "conversation_revision"

// ConversationRevisionData names the edit and the superseded root sequence
// range, from the replaced person's message through its turn's captured tail.
type ConversationRevisionData struct {
	RequestID          string `json:"request_id"`
	Revision           int64  `json:"revision"`
	ReplacesFromSeq    int64  `json:"replaces_from_seq"`
	ReplacesThroughSeq int64  `json:"replaces_through_seq"`
}
