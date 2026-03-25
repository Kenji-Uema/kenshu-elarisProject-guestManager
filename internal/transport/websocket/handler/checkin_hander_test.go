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

func TestCheckinHandlerHandle(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		fakeReception, fakeCleaning, fakeClock, fakeReader, fakeWriter := fakesDependencies(t)
		handler := NewCheckinHandler(fakeReception, fakeClock, fakeWriter, fakeReader)

		now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
		booking := mustBooking(t, "cottage-7")
		requests := make([]dto.SystemRequest, 0, 2)
		notifications := make([]dto.SystemNotification, 0, 2)

		fakeClock.NowFn = func(ctx context.Context) (*time.Time, error) {
			return &now, nil
		}
		waitedActions := make([]dto.GuestAction, 0, 2)
		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			waitedActions = append(waitedActions, action)
			return &dto.ChatMessage{
				MessageId: "guest-checkin",
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: action,
				},
			}, nil
		}
		fakeWriter.SendSystemRequestFn = func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
			requests = append(requests, request)

			switch request {
			case dto.SystemRequest_REQUEST_DOCUMENT:
				if _, ok := responseType.GetPayload().(*dto.GuestResponse_ShowDocument); !ok {
					t.Fatalf("Handle() unexpected document response type: %+v", responseType)
				}
				return &dto.ChatMessage{
					Payload: &dto.ChatMessage_GuestResponse{
						GuestResponse: &dto.GuestResponse{
							Payload: &dto.GuestResponse_ShowDocument{
								ShowDocument: &dto.ShowDocument{DocumentId: "doc-123"},
							},
						},
					},
				}, nil
			case dto.SystemRequest_GIVE_COTTAGE_KEY:
				if _, ok := responseType.GetPayload().(*dto.GuestResponse_ReceiveCottageKey); !ok {
					t.Fatalf("Handle() unexpected cottage key response type: %+v", responseType)
				}
				return &dto.ChatMessage{
					Payload: &dto.ChatMessage_GuestResponse{
						GuestResponse: &dto.GuestResponse{
							Payload: &dto.GuestResponse_ReceiveCottageKey{
								ReceiveCottageKey: &dto.ReceiveCottageKey{CottageKeyId: "key-7"},
							},
						},
					},
				}, nil
			default:
				t.Fatalf("Handle() unexpected system request: %v", request)
				return nil, nil
			}
		}
		fakeWriter.SendSystemNotificationFn = func(ctx context.Context, notification dto.SystemNotification) error {
			notifications = append(notifications, notification)
			return nil
		}
		fakeReception.CheckInFn = func(ctx context.Context, guestDocument string, today time.Time) (domain.Booking, error) {
			if guestDocument != "doc-123" {
				t.Fatalf("Handle() unexpected document: %q", guestDocument)
			}
			if !today.Equal(now) {
				t.Fatalf("Handle() unexpected checkin time: %v", today)
			}
			return booking, nil
		}

		got, err := handler.Handle(context.Background())
		if err != nil {
			t.Fatalf("Handle() unexpected error: %v", err)
		}
		if got.CottageName != booking.CottageName {
			t.Fatalf("Handle() unexpected booking: %+v", got)
		}
		if fakeClock.NowCallCount != 1 {
			t.Fatalf("Handle() expected 1 clock call, got %d", fakeClock.NowCallCount)
		}
		if len(waitedActions) != 2 || waitedActions[0] != dto.GuestAction_SHOW_FOR_CHECKIN || waitedActions[1] != dto.GuestAction_TAKE_COTTAGE_KEY {
			t.Fatalf("Handle() unexpected guest actions: %+v", waitedActions)
		}
		if fakeReception.CheckInCallCount != 1 {
			t.Fatalf("Handle() expected 1 checkin call, got %d", fakeReception.CheckInCallCount)
		}
		if fakeReception.ReceiveCottageKeyCallCount != 1 {
			t.Fatalf("Handle() expected 1 receive cottage key call, got %d", fakeReception.ReceiveCottageKeyCallCount)
		}
		if fakeReception.LastReceiveCottageName != booking.CottageName || fakeReception.LastReceiveKeyNumber != "key-7" {
			t.Fatalf("Handle() unexpected receive cottage key args: room=%q key=%q", fakeReception.LastReceiveCottageName, fakeReception.LastReceiveKeyNumber)
		}
		if fakeReception.CheckinFallbackCallCount != 0 {
			t.Fatalf("Handle() expected no fallback calls, got %d", fakeReception.CheckinFallbackCallCount)
		}
		if fakeCleaning.CleanRoomCallCount != 0 {
			t.Fatalf("Handle() expected no cleaning calls, got %d", fakeCleaning.CleanRoomCallCount)
		}
		if len(requests) != 2 || requests[0] != dto.SystemRequest_REQUEST_DOCUMENT || requests[1] != dto.SystemRequest_GIVE_COTTAGE_KEY {
			t.Fatalf("Handle() unexpected system requests: %+v", requests)
		}
		if len(notifications) != 2 || notifications[0] != dto.SystemNotification_BOOKING_CHECKING || notifications[1] != dto.SystemNotification_CHECK_IN_COMPLETE {
			t.Fatalf("Handle() unexpected notifications: %+v", notifications)
		}
	})

	t.Run("fallback success", func(t *testing.T) {
		fakeReception, fakeCleaning, fakeClock, fakeReader, fakeWriter := fakesDependencies(t)
		handler := NewCheckinHandler(fakeReception, fakeClock, fakeWriter, fakeReader)
		now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
		booking := mustBooking(t, "cottage-8")
		requests := make([]dto.SystemRequest, 0, 3)

		fakeClock.NowFn = func(ctx context.Context) (*time.Time, error) {
			return &now, nil
		}
		waitedActions := make([]dto.GuestAction, 0, 2)
		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			waitedActions = append(waitedActions, action)
			return &dto.ChatMessage{
				MessageId: "guest-checkin",
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: action,
				},
			}, nil
		}
		fakeWriter.SendSystemRequestFn = func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
			requests = append(requests, request)

			switch request {
			case dto.SystemRequest_REQUEST_DOCUMENT:
				return &dto.ChatMessage{
					Payload: &dto.ChatMessage_GuestResponse{
						GuestResponse: &dto.GuestResponse{
							Payload: &dto.GuestResponse_ShowDocument{
								ShowDocument: &dto.ShowDocument{DocumentId: "doc-456"},
							},
						},
					},
				}, nil
			case dto.SystemRequest_REQUEST_BOOKING_NUMBER:
				return &dto.ChatMessage{
					Payload: &dto.ChatMessage_GuestResponse{
						GuestResponse: &dto.GuestResponse{
							Payload: &dto.GuestResponse_ShowBookingNumber{
								ShowBookingNumber: &dto.ShowBookingNumber{BookingId: "booking-42"},
							},
						},
					},
				}, nil
			case dto.SystemRequest_GIVE_COTTAGE_KEY:
				return &dto.ChatMessage{
					Payload: &dto.ChatMessage_GuestResponse{
						GuestResponse: &dto.GuestResponse{
							Payload: &dto.GuestResponse_ReceiveCottageKey{
								ReceiveCottageKey: &dto.ReceiveCottageKey{CottageKeyId: "key-8"},
							},
						},
					},
				}, nil
			default:
				t.Fatalf("Handle() unexpected system request: %v", request)
				return nil, nil
			}
		}
		fakeReception.CheckInFn = func(ctx context.Context, guestDocument string, today time.Time) (domain.Booking, error) {
			return domain.Booking{}, errors.New("default failed")
		}
		fakeReception.CheckinFallbackFn = func(ctx context.Context, document string, bookingNumber string, today time.Time) (domain.Booking, error) {
			if document != "doc-456" || bookingNumber != "booking-42" {
				t.Fatalf("Handle() unexpected fallback args: %q %q", document, bookingNumber)
			}
			return booking, nil
		}

		got, err := handler.Handle(context.Background())
		if err != nil {
			t.Fatalf("Handle() unexpected error: %v", err)
		}
		if got.CottageName != booking.CottageName {
			t.Fatalf("Handle() unexpected booking: %+v", got)
		}
		if len(waitedActions) != 2 || waitedActions[0] != dto.GuestAction_SHOW_FOR_CHECKIN || waitedActions[1] != dto.GuestAction_TAKE_COTTAGE_KEY {
			t.Fatalf("Handle() unexpected guest actions: %+v", waitedActions)
		}
		if fakeReception.CheckInCallCount != 1 || fakeReception.CheckinFallbackCallCount != 1 {
			t.Fatalf("Handle() unexpected reception calls: checkin=%d fallback=%d", fakeReception.CheckInCallCount, fakeReception.CheckinFallbackCallCount)
		}
		if fakeReception.ReceiveCottageKeyCallCount != 1 {
			t.Fatalf("Handle() expected 1 receive cottage key call, got %d", fakeReception.ReceiveCottageKeyCallCount)
		}
		if fakeReception.LastReceiveCottageName != booking.CottageName || fakeReception.LastReceiveKeyNumber != "key-8" {
			t.Fatalf("Handle() unexpected receive cottage key args: room=%q key=%q", fakeReception.LastReceiveCottageName, fakeReception.LastReceiveKeyNumber)
		}
		if fakeCleaning.CleanRoomCallCount != 0 {
			t.Fatalf("Handle() expected no cleaning calls, got %d", fakeCleaning.CleanRoomCallCount)
		}
		if len(requests) != 3 || requests[0] != dto.SystemRequest_REQUEST_DOCUMENT || requests[1] != dto.SystemRequest_REQUEST_BOOKING_NUMBER || requests[2] != dto.SystemRequest_GIVE_COTTAGE_KEY {
			t.Fatalf("Handle() unexpected system requests: %+v", requests)
		}
	})

	t.Run("returns error on malformed document response", func(t *testing.T) {
		fakeReception, fakeCleaning, fakeClock, fakeReader, fakeWriter := fakesDependencies(t)
		handler := NewCheckinHandler(fakeReception, fakeClock, fakeWriter, fakeReader)
		now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)

		fakeClock.NowFn = func(ctx context.Context) (*time.Time, error) {
			return &now, nil
		}
		waitedActions := make([]dto.GuestAction, 0, 1)
		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			waitedActions = append(waitedActions, action)
			return &dto.ChatMessage{
				MessageId: "guest-checkin",
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: action,
				},
			}, nil
		}
		fakeWriter.SendSystemRequestFn = func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
			return &dto.ChatMessage{
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: dto.GuestAction_SHOW_DOCUMENT,
				},
			}, nil
		}

		_, err := handler.Handle(context.Background())
		if !errors.Is(err, ErrUnexpectedResponse) {
			t.Fatalf("Handle() expected ErrUnexpectedResponse, got %v", err)
		}
		if len(waitedActions) != 1 || waitedActions[0] != dto.GuestAction_SHOW_FOR_CHECKIN {
			t.Fatalf("Handle() unexpected guest actions: %+v", waitedActions)
		}
		if fakeReception.CheckInCallCount != 0 || fakeReception.CheckinFallbackCallCount != 0 {
			t.Fatalf("Handle() unexpected reception calls: checkin=%d fallback=%d", fakeReception.CheckInCallCount, fakeReception.CheckinFallbackCallCount)
		}
		if fakeReception.ReceiveCottageKeyCallCount != 0 {
			t.Fatalf("Handle() expected no receive cottage key calls, got %d", fakeReception.ReceiveCottageKeyCallCount)
		}
		if fakeCleaning.CleanRoomCallCount != 0 {
			t.Fatalf("Handle() expected no cleaning calls, got %d", fakeCleaning.CleanRoomCallCount)
		}
	})
}

func fakesDependencies(t *testing.T) (*appfakes.ReceptionService, *appfakes.CleaningService,
	*infrafakes.FakeClockClient, *chatfakes.Reader, *chatfakes.Writer) {

	t.Helper()

	reception := &appfakes.ReceptionService{}
	cleaning := &appfakes.CleaningService{}
	reader := &chatfakes.Reader{}
	writer := &chatfakes.Writer{}
	clockClient := &infrafakes.FakeClockClient{}

	return reception, cleaning, clockClient, reader, writer
}

func mustBooking(t *testing.T, cottage string) domain.Booking {
	t.Helper()

	stayPeriod, err := domain.NewPeriod(
		time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewPeriod() unexpected error: %v", err)
	}

	booking, err := domain.NewBooking([12]byte{1}, [12]byte{}, 2, stayPeriod, cottage, "confirmed")
	if err != nil {
		t.Fatalf("NewBooking() unexpected error: %v", err)
	}

	return booking
}
