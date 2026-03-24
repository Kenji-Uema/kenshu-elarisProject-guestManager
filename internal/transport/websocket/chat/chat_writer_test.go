package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/chatErrors"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat/fakes"
)

func TestWriterSendSystemNotification(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		fakeChat := &fakes.Chat{}
		fakeChat.OnWrite = func(msg *dto.ChatMessage) {
			fakeChat.Messages = append(fakeChat.Messages, ackForTestMessage(msg))
		}

		writer := &writer{chat: fakeChat}
		err := writer.SendSystemNotification(context.Background(), dto.SystemNotification_BREAKFAST_READY)
		if err != nil {
			t.Fatalf("SendSystemNotification() unexpected error: %v", err)
		}

		if len(fakeChat.Written) != 1 {
			t.Fatalf("SendSystemNotification() expected 1 write, got %d", len(fakeChat.Written))
		}
		if fakeChat.Written[0].GetSystemNotification() != dto.SystemNotification_BREAKFAST_READY {
			t.Fatalf("SendSystemNotification() unexpected notification: %v", fakeChat.Written[0].GetSystemNotification())
		}
	})

	t.Run("returns context deadline exceeded when no ack arrives before context expires", func(t *testing.T) {
		fakeChat := &fakes.Chat{WaitForContextDone: true}
		writer := &writer{chat: fakeChat}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err := writer.SendSystemNotification(ctx, dto.SystemNotification_BREAKFAST_READY)
		if err == nil {
			t.Fatal("SendSystemNotification() expected timeout error, got nil")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("SendSystemNotification() expected context deadline exceeded, got %v", err)
		}
	})

	t.Run("resends after ack timeout", func(t *testing.T) {
		ackReadCount := 0
		fakeChat := &fakes.Chat{}
		fakeChat.OnReadAck = func(ctx context.Context) (*dto.ChatMessage, error) {
			ackReadCount++
			if ackReadCount == 1 {
				return nil, &chatErrors.AckNotReceivedErr{Err: context.DeadlineExceeded}
			}
			return fakeChat.ReadOneMessage(ctx)
		}
		fakeChat.OnWrite = func(msg *dto.ChatMessage) {
			if len(fakeChat.Written) == 1 {
				return
			}
			fakeChat.Messages = append(fakeChat.Messages, ackForTestMessage(msg))
		}

		writer := &writer{chat: fakeChat}
		err := writer.SendSystemNotification(context.Background(), dto.SystemNotification_BREAKFAST_READY)
		if err != nil {
			t.Fatalf("SendSystemNotification() unexpected error: %v", err)
		}

		if len(fakeChat.Written) != 2 {
			t.Fatalf("SendSystemNotification() expected 2 writes, got %d", len(fakeChat.Written))
		}
		if fakeChat.Written[0].GetMessageId() != fakeChat.Written[1].GetMessageId() {
			t.Fatalf("SendSystemNotification() expected resend with same message id, got %q and %q", fakeChat.Written[0].GetMessageId(), fakeChat.Written[1].GetMessageId())
		}
	})

	t.Run("ignores unrelated ack while waiting for target ack", func(t *testing.T) {
		fakeChat := &fakes.Chat{}
		fakeChat.OnWrite = func(msg *dto.ChatMessage) {
			fakeChat.Messages = append(fakeChat.Messages, &dto.ChatMessage{
				MessageId:     "ack-other-message",
				CorrelationId: "other-message-id",
				Payload: &dto.ChatMessage_Ack{
					Ack: &dto.Ack{
						AcknowledgedMessageId: "other-message-id",
						Status:                dto.AckStatus_ACK_STATUS_ACCEPTED,
						Code:                  dto.ErrorCode_ERROR_CODE_NONE,
					},
				},
			}, ackForTestMessage(msg))
		}

		writer := &writer{chat: fakeChat}
		err := writer.SendSystemNotification(context.Background(), dto.SystemNotification_BREAKFAST_READY)
		if err != nil {
			t.Fatalf("SendSystemNotification() unexpected error: %v", err)
		}

		if len(fakeChat.Written) != 1 {
			t.Fatalf("SendSystemNotification() expected 1 write, got %d", len(fakeChat.Written))
		}
	})

	t.Run("returns read ack error when it is not an ack timeout", func(t *testing.T) {
		readErr := errors.New("read ack failed")
		fakeChat := &fakes.Chat{ReadErr: readErr}
		writer := &writer{chat: fakeChat}

		err := writer.SendSystemNotification(context.Background(), dto.SystemNotification_BREAKFAST_READY)
		if err == nil {
			t.Fatal("SendSystemNotification() expected error, got nil")
		}
		if !errors.Is(err, readErr) {
			t.Fatalf("SendSystemNotification() expected read error, got %v", err)
		}
		if len(fakeChat.Written) != 1 {
			t.Fatalf("SendSystemNotification() expected 1 write, got %d", len(fakeChat.Written))
		}
	})
}

func TestWriterSendSystemRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns guest response", func(t *testing.T) {
		fakeChat := &fakes.Chat{}
		fakeChat.OnWrite = func(msg *dto.ChatMessage) {
			if msg.GetSystemRequest() != dto.SystemRequest_REQUEST_DOCUMENT {
				return
			}

			newMsg := &dto.ChatMessage{
				MessageId:     "guest-document-response",
				CorrelationId: msg.GetMessageId(),
				Payload: &dto.ChatMessage_GuestResponse{
					GuestResponse: &dto.GuestResponse{
						Payload: &dto.GuestResponse_ShowDocument{
							ShowDocument: &dto.ShowDocument{DocumentId: "doc-123"},
						},
					},
				},
			}

			fakeChat.Messages = append(fakeChat.Messages, ackForTestMessage(msg), newMsg)
		}

		writer := &writer{chat: fakeChat}
		msg, err := writer.SendSystemRequest(context.Background(), dto.SystemRequest_REQUEST_DOCUMENT, &dto.GuestResponse{
			Payload: &dto.GuestResponse_ShowDocument{},
		})
		if err != nil {
			t.Fatalf("SendSystemRequest() unexpected error: %v", err)
		}
		if msg.GetGuestResponse().GetShowDocument().GetDocumentId() != "doc-123" {
			t.Fatalf("SendSystemRequest() unexpected response: %+v", msg)
		}
		if len(fakeChat.Written) != 1 {
			t.Fatalf("SendSystemRequest() expected 1 write, got %d", len(fakeChat.Written))
		}
		if len(fakeChat.Acked) != 1 || fakeChat.Acked[0].GetMessageId() != "guest-document-response" {
			t.Fatalf("SendSystemRequest() unexpected acked messages: %+v", fakeChat.Acked)
		}
	})

	t.Run("resends after unexpected reply", func(t *testing.T) {
		writeCount := 0
		fakeChat := &fakes.Chat{}
		fakeChat.OnWrite = func(msg *dto.ChatMessage) {
			writeCount++
			fakeChat.Messages = append(fakeChat.Messages, ackForTestMessage(msg))
			if writeCount == 1 {
				newMsg := &dto.ChatMessage{
					MessageId:     "guest-wrong-reply",
					CorrelationId: msg.GetMessageId(),
					Payload: &dto.ChatMessage_GuestResponse{
						GuestResponse: &dto.GuestResponse{
							Payload: &dto.GuestResponse_ShowDocument{
								ShowDocument: &dto.ShowDocument{DocumentId: "doc-wrong"},
							},
						},
					},
				}

				fakeChat.Messages = append(fakeChat.Messages, newMsg)
				return
			}

			newMsg := &dto.ChatMessage_GuestResponse{
				GuestResponse: &dto.GuestResponse{
					Payload: &dto.GuestResponse_ShowBookingNumber{
						ShowBookingNumber: &dto.ShowBookingNumber{BookingId: "booking-42"},
					},
				},
			}

			fakeChat.Messages = append(fakeChat.Messages, &dto.ChatMessage{
				MessageId:     "guest-booking-response",
				CorrelationId: msg.GetMessageId(),
				Payload:       newMsg,
			})
		}

		writer := &writer{chat: fakeChat}
		msg, err := writer.SendSystemRequest(context.Background(), dto.SystemRequest_REQUEST_BOOKING_NUMBER, &dto.GuestResponse{
			Payload: &dto.GuestResponse_ShowBookingNumber{},
		})
		if err != nil {
			t.Fatalf("SendSystemRequest() unexpected error: %v", err)
		}
		if msg.GetGuestResponse().GetShowBookingNumber().GetBookingId() != "booking-42" {
			t.Fatalf("SendSystemRequest() unexpected response: %+v", msg)
		}
		if len(fakeChat.Written) != 2 {
			t.Fatalf("SendSystemRequest() expected 2 writes, got %d", len(fakeChat.Written))
		}
		if fakeChat.Written[0].GetMessageId() != fakeChat.Written[1].GetMessageId() {
			t.Fatalf("SendSystemRequest() expected resend with same message id, got %q and %q", fakeChat.Written[0].GetMessageId(), fakeChat.Written[1].GetMessageId())
		}
		if len(fakeChat.Acked) != 2 {
			t.Fatalf("SendSystemRequest() expected 2 acked guest messages, got %d", len(fakeChat.Acked))
		}
	})

	t.Run("resends after ack timeout", func(t *testing.T) {
		ackReadCount := 0
		fakeChat := &fakes.Chat{}
		fakeChat.OnReadAck = func(ctx context.Context) (*dto.ChatMessage, error) {
			ackReadCount++
			if ackReadCount == 1 {
				return nil, &chatErrors.AckNotReceivedErr{Err: context.DeadlineExceeded}
			}
			return fakeChat.ReadOneMessage(ctx)
		}
		fakeChat.OnWrite = func(msg *dto.ChatMessage) {
			if len(fakeChat.Written) == 1 {
				return
			}

			fakeChat.Messages = append(fakeChat.Messages, ackForTestMessage(msg), &dto.ChatMessage{
				MessageId:     "guest-document-response",
				CorrelationId: msg.GetMessageId(),
				Payload: &dto.ChatMessage_GuestResponse{
					GuestResponse: &dto.GuestResponse{
						Payload: &dto.GuestResponse_ShowDocument{
							ShowDocument: &dto.ShowDocument{DocumentId: "doc-123"},
						},
					},
				},
			})
		}

		writer := &writer{chat: fakeChat}
		msg, err := writer.SendSystemRequest(context.Background(), dto.SystemRequest_REQUEST_DOCUMENT, &dto.GuestResponse{
			Payload: &dto.GuestResponse_ShowDocument{},
		})
		if err != nil {
			t.Fatalf("SendSystemRequest() unexpected error: %v", err)
		}

		if msg.GetGuestResponse().GetShowDocument().GetDocumentId() != "doc-123" {
			t.Fatalf("SendSystemRequest() unexpected response: %+v", msg)
		}
		if len(fakeChat.Written) != 2 {
			t.Fatalf("SendSystemRequest() expected 2 writes, got %d", len(fakeChat.Written))
		}
		if fakeChat.Written[0].GetMessageId() != fakeChat.Written[1].GetMessageId() {
			t.Fatalf("SendSystemRequest() expected resend with same message id, got %q and %q", fakeChat.Written[0].GetMessageId(), fakeChat.Written[1].GetMessageId())
		}
		if len(fakeChat.Acked) != 1 || fakeChat.Acked[0].GetMessageId() != "guest-document-response" {
			t.Fatalf("SendSystemRequest() unexpected acked messages: %+v", fakeChat.Acked)
		}
	})

	t.Run("returns context deadline exceeded when no ack arrives before context expires", func(t *testing.T) {
		fakeChat := &fakes.Chat{WaitForContextDone: true}
		writer := &writer{chat: fakeChat}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_DOCUMENT, &dto.GuestResponse{
			Payload: &dto.GuestResponse_ShowDocument{},
		})
		if err == nil {
			t.Fatal("SendSystemRequest() expected timeout error, got nil")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("SendSystemRequest() expected context deadline exceeded, got %v", err)
		}
	})
}

func ackForTestMessage(msg *dto.ChatMessage) *dto.ChatMessage {
	return &dto.ChatMessage{
		MessageId:     "ack-" + msg.GetMessageId(),
		CorrelationId: msg.GetMessageId(),
		Payload: &dto.ChatMessage_Ack{
			Ack: &dto.Ack{
				AcknowledgedMessageId: msg.GetMessageId(),
				Status:                dto.AckStatus_ACK_STATUS_ACCEPTED,
				Code:                  dto.ErrorCode_ERROR_CODE_NONE,
			},
		},
	}
}
