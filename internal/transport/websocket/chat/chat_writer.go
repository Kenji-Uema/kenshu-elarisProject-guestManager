package chat

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/chatErrors"
	"github.com/google/uuid"
)

var errUnexpectedGuestResponse = errors.New("unexpected guest response")

type Writer interface {
	SendSystemRequest(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error)
	SendSystemNotification(ctx context.Context, notification dto.SystemNotification) error
}

type writer struct {
	chat Chat
}

func (w *writer) SendSystemRequest(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
	requestID := uuid.NewString()
	message := &dto.ChatMessage{
		MessageId:       requestID,
		CorrelationId:   requestID,
		Sender:          dto.Sender_SENDER_SYSTEM,
		ProtocolVersion: "lodging.v1",
		Payload: &dto.ChatMessage_SystemRequest{
			SystemRequest: request,
		},
	}

	for {
		if err := w.chat.WriteOneMessage(ctx, message); err != nil {
			return nil, err
		}

		if err := w.waitForAck(ctx, message.GetMessageId()); err != nil {
			var ackErr *chatErrors.AckNotReceivedErr
			if errors.As(err, &ackErr) {
				slog.DebugContext(ctx, "resending websocket system request after ack timeout",
					"message_id", message.GetMessageId())
				continue
			}
			return nil, err
		}

		resp, err := w.waitForReply(ctx, message.GetMessageId(), responseType)
		if err == nil {
			return resp, nil
		}

		if errors.Is(err, errUnexpectedGuestResponse) {
			slog.DebugContext(ctx, "resending websocket system request after unexpected guest response",
				"message_id", message.GetMessageId())
			continue
		}

		return nil, err
	}
}

func (w *writer) SendSystemNotification(ctx context.Context, notification dto.SystemNotification) error {
	message := &dto.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   uuid.NewString(),
		Sender:          dto.Sender_SENDER_SYSTEM,
		ProtocolVersion: "lodging.v1",
		Payload: &dto.ChatMessage_SystemNotification{
			SystemNotification: notification,
		},
	}

	return w.chat.WriteOneMessage(ctx, message)
}

func (w *writer) waitForReply(ctx context.Context, correlationID string, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
	for {
		msg, err := w.chat.ReadReply(ctx)
		if err != nil {
			return nil, err
		}

		if msg.GetAck() != nil {
			slog.DebugContext(ctx, "ignoring websocket ack while waiting for system reply",
				"correlation_id", correlationID,
				"message_id", msg.GetMessageId())
			continue
		}

		w.chat.SendAck(ctx, msg)
		if msg.GetCorrelationId() != correlationID {
			slog.DebugContext(ctx, "ignoring websocket message while waiting for system reply",
				"expected_correlation_id", correlationID,
				"received_correlation_id", msg.GetCorrelationId(),
				"message_id", msg.GetMessageId())
			continue
		}

		if guestResponseMatchesExpected(msg.GetGuestResponse(), responseType) {
			return msg, nil
		}

		slog.DebugContext(ctx, "received unexpected guest response for system request",
			"correlation_id", correlationID,
			"message_id", msg.GetMessageId())
		return nil, errUnexpectedGuestResponse
	}
}

func guestResponseMatchesExpected(actual *dto.GuestResponse, expected *dto.GuestResponse) bool {
	if actual == nil || expected == nil {
		return false
	}

	switch expected.GetPayload().(type) {
	case *dto.GuestResponse_ShowDocument:
		return actual.GetShowDocument() != nil
	case *dto.GuestResponse_ShowBookingNumber:
		return actual.GetShowBookingNumber() != nil
	case *dto.GuestResponse_ReceiveCottageKey:
		return actual.GetReceiveCottageKey() != nil
	case *dto.GuestResponse_ReturnCottageKey:
		return actual.GetReturnCottageKey() != nil
	default:
		return false
	}
}

func (w *writer) waitForAck(ctx context.Context, messageID string) error {
	for {
		msg, err := w.chat.ReadAck(ctx)
		if err != nil {
			return err
		}

		if ack := msg.GetAck(); ack != nil {
			if ack.GetAcknowledgedMessageId() == messageID {
				return nil
			}

			slog.DebugContext(ctx, "wrong ack",
				"expected_message_id", messageID,
				"acknowledged_message_id", ack.GetAcknowledgedMessageId())
			continue
		}

		slog.DebugContext(ctx, "ignoring guest message while waiting for system ack",
			"expected_message_id", messageID,
			"received_action", msg.GetGuestAction().String(),
			"message_id", msg.GetMessageId())
	}
}
