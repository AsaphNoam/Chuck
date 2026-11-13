package runtime

import (
	"context"
)

func (r *Registry) SetMessageOutcomeHandler(fn func(string, string, string) error) {
	r.chat.SetMessageOutcomeHandler(fn)
}

// StartLaunchedTask sends the initial assignment together with any input
// retained for this owner, using the ordinary chat turn gate.
func (r *Registry) StartLaunchedTask(id, instruction string) error {
	rt, err := r.ownerFor(id)
	if err != nil {
		return err
	}
	chat, ok := rt.(*ChatRuntime)
	if !ok {
		return ErrNoHandle
	}
	return chat.sendPrompt(id, LaunchedTaskPrompt(instruction), true)
}
func (r *Registry) SendMessageOrHold(ctx context.Context, id string, m Message) (bool, error) {
	rt, err := r.ownerFor(id)
	if err != nil {
		return false, err
	}
	if chat, ok := rt.(*ChatRuntime); ok {
		return chat.SendMessageOrHold(ctx, id, m)
	}
	if len(m.Attachments) > 0 {
		return false, NewAPIError(CodeAttachmentCapability, "Attachments require a chat agent")
	}
	return false, rt.SendPrompt(ctx, id, m.Text)
}
func (r *Registry) SteerMessage(ctx context.Context, id string, m Message) (SteerOutcome, error) {
	rt, err := r.ownerFor(id)
	if err != nil {
		return "", err
	}
	if chat, ok := rt.(*ChatRuntime); ok {
		return chat.SteerMessage(ctx, id, m)
	}
	return "", ErrSteeringUnsupported
}
func (r *Registry) HeldMessage(id string) (Message, int64, error) {
	m, seq, _, err := r.HeldSnapshot(id)
	return m, seq, err
}

func (r *Registry) HeldSnapshot(id string) (Message, int64, string, error) {
	rt, err := r.ownerFor(id)
	if err != nil {
		return Message{}, 0, "", err
	}
	if chat, ok := rt.(*ChatRuntime); ok {
		return chat.HeldSnapshot(id)
	}
	return Message{}, 0, "", nil
}
