package chat

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

type Reader interface {
	WaitForGuestAction(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error)
	AckGuestAction(ctx context.Context) error
}

type reader struct {
	chat Chat

	mu      sync.Mutex
	pending []*dto.ChatMessage
}

func (r *reader) WaitForGuestAction(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
	if msg := r.dequeueGuestAction(action); msg != nil {
		return msg, nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		msg, err := r.chat.ReadOneMessage(ctx)
		if err != nil {
			return nil, err
		}

		r.chat.SendAck(ctx, msg)

		if msg.GetGuestAction() == action {
			return msg, nil
		}

		if msg.GetGuestAction() != dto.GuestAction_GUEST_ACTION_UNSPECIFIED {
			r.enqueuePending(msg)
		}

		slog.DebugContext(
			ctx,
			"ignoring websocket message while waiting for guest action",
			"expected_action", action.String(),
			"received_action", msg.GetGuestAction().String(),
			"message_id", msg.GetMessageId(),
		)
	}
}

func (r *reader) AckGuestAction(ctx context.Context) error {
	msg, err := r.chat.ReadOneMessage(ctx)
	if err != nil {
		return err
	}

	slog.DebugContext(
		ctx, "action received",
		"received_action", msg.GetGuestAction().String(),
		"message_id", msg.GetMessageId(),
	)

	r.chat.SendAck(ctx, msg)

	return nil
}

func (r *reader) dequeueGuestAction(action dto.GuestAction) *dto.ChatMessage {
	r.mu.Lock()
	defer r.mu.Unlock()

	for idx, msg := range r.pending {
		if msg.GetGuestAction() != action {
			continue
		}

		r.pending = append(r.pending[:idx], r.pending[idx+1:]...)
		return msg
	}

	return nil
}

func (r *reader) enqueuePending(msg *dto.ChatMessage) {
	if msg == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending = append(r.pending, msg)
}
