package message

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Reader struct {
	exchange *exchange
}

func (r *Reader) WaitForGuestAction(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		msg, err := r.exchange.read(ctx)
		if err != nil {
			return nil, err
		}

		r.sendAck(ctx, msg)

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

func (r *Reader) Read(ctx context.Context) {
	for {
		msg, err := r.exchange.read(ctx)
		if err != nil {
			slog.WarnContext(ctx, "websocket read", "error", err)
			return
		}

		r.sendAck(ctx, msg)

		correlationID := msg.GetCorrelationId()
		if correlationID == "" && msg.GetAck() != nil {
			correlationID = msg.GetAck().GetAcknowledgedMessageId()
		}

		reply := r.exchange.getPendingReply(correlationID)

		if reply == nil {
			slog.WarnContext(ctx, "websocket message dropped: no receiver", "correlation_id", correlationID)
			continue
		}

		target := reply.replyCh
		route := "reply"
		if msg.GetAck() != nil {
			target = reply.ackCh
			route = "ack"
		}

		select {
		case target <- msg:
			slog.DebugContext(ctx, "websocket message delivered", "route", route, "correlation_id", correlationID, "message_id", msg.GetMessageId())
		default:
			slog.WarnContext(
				ctx,
				"websocket message dropped: receiver channel full",
				"route", route,
				"correlation_id", correlationID,
				"message_id", msg.GetMessageId(),
			)
		}
	}
}

func (r *Reader) sendAck(ctx context.Context, msg *dto.ChatMessage) {
	if msg == nil || msg.GetAck() != nil {
		return
	}

	ack := &dto.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   msg.GetMessageId(),
		SentAt:          timestamppb.Now(),
		Sender:          dto.Sender_SENDER_SYSTEM,
		ProtocolVersion: msg.GetProtocolVersion(),
		Phase:           msg.GetPhase(),
		Payload: &dto.ChatMessage_Ack{
			Ack: &dto.Ack{
				AcknowledgedMessageId: msg.GetMessageId(),
				Status:                dto.AckStatus_ACK_STATUS_ACCEPTED,
				Code:                  dto.ErrorCode_ERROR_CODE_NONE,
			},
		},
	}

	if err := r.exchange.write(ctx, ack); err != nil {
		slog.WarnContext(ctx, "websocket ack write", "error", err, "message_id", msg.GetMessageId())
	}
}
