package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat/fakes"
)

func TestReaderWaitForGuestAction(t *testing.T) {
	t.Parallel()

	t.Run("ignores unexpected messages until match", func(t *testing.T) {
		fakeChat := &fakes.Chat{
			Messages: []*dto.ChatMessage{
				{
					MessageId: "guest-go-dinner",
					Payload: &dto.ChatMessage_GuestAction{
						GuestAction: dto.GuestAction_GO_FOR_DINNER,
					},
				},
				{
					MessageId: "guest-checkin",
					Payload: &dto.ChatMessage_GuestAction{
						GuestAction: dto.GuestAction_SHOW_FOR_CHECKIN,
					},
				},
			},
		}

		reader := &reader{chat: fakeChat}
		msg, err := reader.WaitForGuestAction(context.Background(), dto.GuestAction_SHOW_FOR_CHECKIN)
		if err != nil {
			t.Fatalf("WaitForGuestAction() unexpected error: %v", err)
		}
		if msg.GetMessageId() != "guest-checkin" {
			t.Fatalf("WaitForGuestAction() unexpected message: %+v", msg)
		}

		if len(fakeChat.Acked) != 2 {
			t.Fatalf("WaitForGuestAction() expected 2 acked messages, got %d", len(fakeChat.Acked))
		}
		if fakeChat.Acked[0].GetMessageId() != "guest-go-dinner" {
			t.Fatalf("WaitForGuestAction() unexpected first acked message: %+v", fakeChat.Acked[0])
		}
		if fakeChat.Acked[1].GetMessageId() != "guest-checkin" {
			t.Fatalf("WaitForGuestAction() unexpected second acked message: %+v", fakeChat.Acked[1])
		}
	})

	t.Run("buffers unmatched guest actions for later waits", func(t *testing.T) {
		fakeChat := &fakes.Chat{
			Messages: []*dto.ChatMessage{
				{
					MessageId: "guest-checkout",
					Payload: &dto.ChatMessage_GuestAction{
						GuestAction: dto.GuestAction_PROCEED_TO_CHECKOUT,
					},
				},
				{
					MessageId: "guest-leave",
					Payload: &dto.ChatMessage_GuestAction{
						GuestAction: dto.GuestAction_LEAVE_COTTAGE,
					},
				},
			},
		}

		reader := &reader{chat: fakeChat}
		msg, err := reader.WaitForGuestAction(context.Background(), dto.GuestAction_LEAVE_COTTAGE)
		if err != nil {
			t.Fatalf("WaitForGuestAction() unexpected error: %v", err)
		}
		if msg.GetMessageId() != "guest-leave" {
			t.Fatalf("WaitForGuestAction() unexpected first message: %+v", msg)
		}

		msg, err = reader.WaitForGuestAction(context.Background(), dto.GuestAction_PROCEED_TO_CHECKOUT)
		if err != nil {
			t.Fatalf("WaitForGuestAction() unexpected error on buffered action: %v", err)
		}
		if msg.GetMessageId() != "guest-checkout" {
			t.Fatalf("WaitForGuestAction() unexpected buffered message: %+v", msg)
		}

		if len(fakeChat.Acked) != 2 {
			t.Fatalf("WaitForGuestAction() expected 2 acked messages, got %d", len(fakeChat.Acked))
		}
	})

	t.Run("returns immediate match", func(t *testing.T) {
		fakeChat := &fakes.Chat{
			Messages: []*dto.ChatMessage{
				{
					MessageId: "guest-checkin",
					Payload: &dto.ChatMessage_GuestAction{
						GuestAction: dto.GuestAction_SHOW_FOR_CHECKIN,
					},
				},
			},
		}

		reader := &reader{chat: fakeChat}
		msg, err := reader.WaitForGuestAction(context.Background(), dto.GuestAction_SHOW_FOR_CHECKIN)
		if err != nil {
			t.Fatalf("WaitForGuestAction() unexpected error: %v", err)
		}
		if msg.GetMessageId() != "guest-checkin" {
			t.Fatalf("WaitForGuestAction() unexpected message: %+v", msg)
		}
		if len(fakeChat.Acked) != 1 || fakeChat.Acked[0].GetMessageId() != "guest-checkin" {
			t.Fatalf("WaitForGuestAction() unexpected acked messages: %+v", fakeChat.Acked)
		}
	})

	t.Run("returns read error", func(t *testing.T) {
		readErr := errors.New("read failed")
		fakeChat := &fakes.Chat{ReadErr: readErr}

		reader := &reader{chat: fakeChat}
		_, err := reader.WaitForGuestAction(context.Background(), dto.GuestAction_SHOW_FOR_CHECKIN)
		if !errors.Is(err, readErr) {
			t.Fatalf("WaitForGuestAction() expected read error, got %v", err)
		}
		if len(fakeChat.Acked) != 0 {
			t.Fatalf("WaitForGuestAction() expected no acks on read error, got %+v", fakeChat.Acked)
		}
	})

	t.Run("returns context error before read", func(t *testing.T) {
		fakeChat := &fakes.Chat{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		reader := &reader{chat: fakeChat}
		_, err := reader.WaitForGuestAction(ctx, dto.GuestAction_SHOW_FOR_CHECKIN)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForGuestAction() expected context cancellation, got %v", err)
		}
		if len(fakeChat.Acked) != 0 {
			t.Fatalf("WaitForGuestAction() expected no acks after context cancellation, got %+v", fakeChat.Acked)
		}
	})

	t.Run("ignores ack messages", func(t *testing.T) {
		fakeChat := &fakes.Chat{
			Messages: []*dto.ChatMessage{
				{
					MessageId: "ack-1",
					Payload: &dto.ChatMessage_Ack{
						Ack: &dto.Ack{
							AcknowledgedMessageId: "system-message",
							Status:                dto.AckStatus_ACK_STATUS_ACCEPTED,
							Code:                  dto.ErrorCode_ERROR_CODE_NONE,
						},
					},
				},
				{
					MessageId: "guest-checkin",
					Payload: &dto.ChatMessage_GuestAction{
						GuestAction: dto.GuestAction_SHOW_FOR_CHECKIN,
					},
				},
			},
		}

		reader := &reader{chat: fakeChat}
		msg, err := reader.WaitForGuestAction(context.Background(), dto.GuestAction_SHOW_FOR_CHECKIN)
		if err != nil {
			t.Fatalf("WaitForGuestAction() unexpected error: %v", err)
		}
		if msg.GetMessageId() != "guest-checkin" {
			t.Fatalf("WaitForGuestAction() unexpected message: %+v", msg)
		}
		if len(fakeChat.Acked) != 1 || fakeChat.Acked[0].GetMessageId() != "guest-checkin" {
			t.Fatalf("WaitForGuestAction() unexpected acked messages: %+v", fakeChat.Acked)
		}
	})
}

func TestReaderAckGuestAction(t *testing.T) {
	t.Parallel()

	fakeChat := &fakes.Chat{
		Messages: []*dto.ChatMessage{
			{
				MessageId: "guest-dinner",
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: dto.GuestAction_GO_FOR_DINNER,
				},
			},
			{
				MessageId: "guest-checkin",
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: dto.GuestAction_SHOW_FOR_CHECKIN,
				},
			},
		},
	}

	reader := &reader{chat: fakeChat}
	err := reader.AckGuestAction(context.Background())
	if err != nil {
		t.Fatalf("AckGuestAction() unexpected error: %v", err)
	}
	if len(fakeChat.Acked) != 1 || fakeChat.Acked[0].GetMessageId() != "guest-dinner" {
		t.Fatalf("AckGuestAction() unexpected acked messages: %+v", fakeChat.Acked)
	}
}
