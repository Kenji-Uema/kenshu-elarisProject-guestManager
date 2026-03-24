package chat

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

type Reader interface {
	WaitForGuestAction(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error)
	AckGuestAction(ctx context.Context) error
}

type reader struct {
	chat Chat
}

func (r *reader) WaitForGuestAction(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
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
