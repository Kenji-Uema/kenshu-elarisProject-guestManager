package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	appfakes "github.com/Kenji-Uema/guestManager/internal/app/fakes"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	infrafakes "github.com/Kenji-Uema/guestManager/internal/infra/clock/fakes"
	chatfakes "github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat/fakes"
)

func TestCheckoutHandlerHandle(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		fakeReception := &appfakes.ReceptionService{}
		fakeClock := &infrafakes.FakeClockClient{}
		fakeReader := &chatfakes.Reader{}
		fakeWriter := &chatfakes.Writer{}
		handler := NewCheckoutHandler(fakeReception, fakeClock, fakeWriter, fakeReader)

		now := time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC)
		booking := mustBooking(t, "cottage-9")
		notifications := make([]dto.SystemNotification, 0, 1)
		waitedActions := make([]dto.GuestAction, 0, 2)

		fakeClock.NowFn = func(ctx context.Context) (*time.Time, error) {
			return &now, nil
		}
		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			waitedActions = append(waitedActions, action)
			return &dto.ChatMessage{
				MessageId: "guest-checkout",
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: action,
				},
			}, nil
		}
		fakeWriter.SendSystemRequestFn = func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
			if request != dto.SystemRequest_REQUEST_COTTAGE_KEY {
				t.Fatalf("Handle() unexpected system request: %v", request)
			}
			if _, ok := responseType.GetPayload().(*dto.GuestResponse_ReturnCottageKey); !ok {
				t.Fatalf("Handle() unexpected cottage key response type: %+v", responseType)
			}

			return &dto.ChatMessage{
				Payload: &dto.ChatMessage_GuestResponse{
					GuestResponse: &dto.GuestResponse{
						Payload: &dto.GuestResponse_ReturnCottageKey{
							ReturnCottageKey: &dto.ReturnCottageKey{CottageKeyId: "key-9"},
						},
					},
				},
			}, nil
		}
		fakeWriter.SendSystemNotificationFn = func(ctx context.Context, notification dto.SystemNotification) error {
			notifications = append(notifications, notification)
			return nil
		}
		fakeReception.CheckOutFn = func(ctx context.Context, gotBooking domain.Booking, today time.Time) error {
			if gotBooking.Id != booking.Id {
				t.Fatalf("Handle() unexpected booking: %+v", gotBooking)
			}
			if !today.Equal(now) {
				t.Fatalf("Handle() unexpected checkout time: %v", today)
			}
			return nil
		}

		err := handler.Handle(context.Background(), booking)
		if err != nil {
			t.Fatalf("Handle() unexpected error: %v", err)
		}
		if fakeClock.NowCallCount != 1 {
			t.Fatalf("Handle() expected 1 clock call, got %d", fakeClock.NowCallCount)
		}
		if fakeReader.WaitForGuestActionCallCount != 2 {
			t.Fatalf("Handle() expected 2 guest action waits, got %d", fakeReader.WaitForGuestActionCallCount)
		}
		if len(waitedActions) != 2 || waitedActions[0] != dto.GuestAction_PROCEED_TO_CHECKOUT || waitedActions[1] != dto.GuestAction_RETURN_COTTAGE_KEY {
			t.Fatalf("Handle() unexpected guest actions: %+v", waitedActions)
		}
		if fakeWriter.SendSystemRequestCallCount != 1 {
			t.Fatalf("Handle() expected 1 system request, got %d", fakeWriter.SendSystemRequestCallCount)
		}
		if fakeReception.CheckOutCallCount != 1 {
			t.Fatalf("Handle() expected 1 checkout call, got %d", fakeReception.CheckOutCallCount)
		}
		if fakeReception.ReturnCottageKeyCallCount != 1 {
			t.Fatalf("Handle() expected 1 return cottage key call, got %d", fakeReception.ReturnCottageKeyCallCount)
		}
		if fakeReception.LastReturnCottageName != booking.CottageName || fakeReception.LastReturnKeyNumber != "key-9" {
			t.Fatalf("Handle() unexpected return cottage key args: room=%q key=%q", fakeReception.LastReturnCottageName, fakeReception.LastReturnKeyNumber)
		}
		if len(notifications) != 1 || notifications[0] != dto.SystemNotification_CHECK_OUT_COMPLETE {
			t.Fatalf("Handle() unexpected notifications: %+v", notifications)
		}
	})

	t.Run("returns error on malformed cottage key response", func(t *testing.T) {
		handler := NewCheckoutHandler(
			&appfakes.ReceptionService{},
			&infrafakes.FakeClockClient{NowFn: func(ctx context.Context) (*time.Time, error) {
				now := time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC)
				return &now, nil
			}},
			&chatfakes.Writer{
				SendSystemRequestFn: func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
					return &dto.ChatMessage{
						Payload: &dto.ChatMessage_GuestAction{
							GuestAction: dto.GuestAction_RETURN_COTTAGE_KEY,
						},
					}, nil
				},
			},
			&chatfakes.Reader{
				WaitForGuestActionFn: func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
					return &dto.ChatMessage{
						Payload: &dto.ChatMessage_GuestAction{
							GuestAction: dto.GuestAction_PROCEED_TO_CHECKOUT,
						},
					}, nil
				},
			},
		)

		err := handler.Handle(context.Background(), mustBooking(t, "cottage-10"))
		if !errors.Is(err, ErrUnexpectedResponse) {
			t.Fatalf("Handle() expected ErrUnexpectedResponse, got %v", err)
		}
	})

	t.Run("returns clock error", func(t *testing.T) {
		wantErr := errors.New("clock failed")
		handler := NewCheckoutHandler(
			&appfakes.ReceptionService{},
			&infrafakes.FakeClockClient{
				NowFn: func(ctx context.Context) (*time.Time, error) {
					return nil, wantErr
				},
			},
			&chatfakes.Writer{},
			&chatfakes.Reader{},
		)

		err := handler.Handle(context.Background(), mustBooking(t, "cottage-11"))
		if !errors.Is(err, wantErr) {
			t.Fatalf("Handle() expected clock error, got %v", err)
		}
	})
}

func TestCheckoutHandlerRequestCottageKey(t *testing.T) {
	t.Parallel()

	t.Run("returns cottage key id", func(t *testing.T) {
		fakeWriter := &chatfakes.Writer{}
		handler := CheckoutHandler{writer: fakeWriter}

		fakeWriter.SendSystemRequestFn = func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
			if request != dto.SystemRequest_REQUEST_COTTAGE_KEY {
				t.Fatalf("requestCottageKey() unexpected system request: %v", request)
			}
			if _, ok := responseType.GetPayload().(*dto.GuestResponse_ReturnCottageKey); !ok {
				t.Fatalf("requestCottageKey() unexpected response type: %+v", responseType)
			}

			return &dto.ChatMessage{
				Payload: &dto.ChatMessage_GuestResponse{
					GuestResponse: &dto.GuestResponse{
						Payload: &dto.GuestResponse_ReturnCottageKey{
							ReturnCottageKey: &dto.ReturnCottageKey{CottageKeyId: "key-123"},
						},
					},
				},
			}, nil
		}

		_, got, err := handler.requestCottageKey(context.Background())
		if err != nil {
			t.Fatalf("requestCottageKey() unexpected error: %v", err)
		}
		if got != "key-123" {
			t.Fatalf("requestCottageKey() key = %q, want %q", got, "key-123")
		}
	})

	t.Run("returns unexpected response error", func(t *testing.T) {
		handler := CheckoutHandler{
			writer: &chatfakes.Writer{
				SendSystemRequestFn: func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
					return &dto.ChatMessage{
						Payload: &dto.ChatMessage_GuestAction{
							GuestAction: dto.GuestAction_RETURN_COTTAGE_KEY,
						},
					}, nil
				},
			},
		}

		_, _, err := handler.requestCottageKey(context.Background())
		if !errors.Is(err, ErrUnexpectedResponse) {
			t.Fatalf("requestCottageKey() expected ErrUnexpectedResponse, got %v", err)
		}
	})
}
