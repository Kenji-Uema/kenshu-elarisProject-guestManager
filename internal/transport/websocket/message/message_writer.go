package message

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/chatErrors"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Writer struct {
	exchange *exchange
}

func (w *Writer) SendSystemRequest(ctx context.Context, request dto.SystemRequest, responseType dto.GuestAction) (*dto.ChatMessage, error) {
	requestID := uuid.NewString()
	message := &dto.ChatMessage{
		MessageId:       requestID,
		CorrelationId:   requestID,
		SentAt:          timestamppb.Now(),
		Sender:          dto.Sender_SENDER_SYSTEM,
		ProtocolVersion: "lodging.v1",
		Phase:           dto.LifecyclePhase_LIFECYCLE_PHASE_CHECK_IN,
		Payload: &dto.ChatMessage_SystemRequest{
			SystemRequest: request,
		},
	}

	resp, err := w.sendWaitAckAndReply(ctx, message, 10*time.Second, 10*time.Second)
	if err != nil {
		return nil, err
	}
	if resp.GetGuestAction() != responseType {
		return nil, fmt.Errorf("unexpected guest action: %s", resp.GetGuestAction())
	}

	return resp, nil
}

func (w *Writer) SendSystemNotification(ctx context.Context, notification dto.SystemNotification) error {
	message := &dto.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   uuid.NewString(),
		SentAt:          timestamppb.Now(),
		Sender:          dto.Sender_SENDER_SYSTEM,
		ProtocolVersion: "lodging.v1",
		Payload: &dto.ChatMessage_SystemNotification{
			SystemNotification: notification,
		},
	}

	return w.SendAndWaitAck(ctx, message, 10*time.Second)
}

func (w *Writer) SendAndWaitAck(ctx context.Context, msg *dto.ChatMessage, timeout time.Duration) error {
	reply, messageID, err := w.exchange.registerMessage(msg)
	if err != nil {
		return err
	}
	defer w.exchange.unregisterMessage(messageID)

	if err := w.exchange.write(ctx, msg); err != nil {
		return err
	}

	return w.waitForAck(reply, timeout)
}

func (w *Writer) sendWaitAckAndReply(ctx context.Context, msg *dto.ChatMessage, ackTimeout, replyTimeout time.Duration) (*dto.ChatMessage, error) {
	reply, messageID, err := w.exchange.registerMessage(msg)
	if err != nil {
		return nil, err
	}
	defer w.exchange.unregisterMessage(messageID)

	if err := w.exchange.write(ctx, msg); err != nil {
		return nil, err
	}

	if err := w.waitForAck(reply, ackTimeout); err != nil {
		return nil, err
	}

	return w.waitForReply(reply, replyTimeout)
}

func (w *Writer) waitForAck(reply *inflightRequest, timeout time.Duration) error {
	select {
	case <-reply.ackCh:
		return nil
	case <-time.After(timeout):
		return &chatErrors.AckNotReceivedErr{Err: context.DeadlineExceeded}
	}
}

func (w *Writer) waitForReply(reply *inflightRequest, timeout time.Duration) (*dto.ChatMessage, error) {
	select {
	case resp := <-reply.replyCh:
		return resp, nil
	case <-time.After(timeout):
		return nil, &chatErrors.ReplyNotReceivedErr{Err: context.DeadlineExceeded}
	}
}
