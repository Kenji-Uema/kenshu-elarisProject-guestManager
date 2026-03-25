package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	appfakes "github.com/Kenji-Uema/guestManager/internal/app/fakes"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	mqfakes "github.com/Kenji-Uema/guestManager/internal/infra/mq/fakes"
	chatfakes "github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat/fakes"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStayHandlerHandle(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		booking := mustBooking(t, "cottage-7")
		stayPeriod, err := domain.NewPeriod(
			time.Date(2026, 3, 22, 15, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("NewPeriod() error = %v", err)
		}
		booking.StayPeriod = stayPeriod

		fakeCleaning := &appfakes.CleaningService{}
		fakeWriter := &chatfakes.Writer{}
		fakeReader := &chatfakes.Reader{}
		fakeNotifications := &fakeNotificationService{}
		timeEvents, dayDeliveries, stopTimeEvents := newTestTimeEventService(t)
		defer stopTimeEvents()

		handler := NewStayHandler(fakeNotifications, fakeCleaning, timeEvents, fakeWriter, fakeReader)

		waitedActions := make([]dto.GuestAction, 0, 3)
		ackedActions := 0
		notifications := make([]dto.SystemNotification, 0, 3)
		secondDayEventSent := false

		fakeReader.AckGuestActionFn = func(ctx context.Context) error {
			ackedActions++
			return nil
		}
		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			waitedActions = append(waitedActions, action)
			if action == dto.GuestAction_GO_FOR_DINNER && len(waitedActions) == 3 && !secondDayEventSent {
				secondDayEventSent = true
				dayDeliveries <- amqp.Delivery{
					Acknowledger: &mqfakes.FakeAcknowledger{},
					DeliveryTag:  2,
					Body:         mustMarshalTimeEventStay(t, time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)),
				}
			}
			return &dto.ChatMessage{
				MessageId: "guest-action",
				Payload: &dto.ChatMessage_GuestAction{
					GuestAction: action,
				},
			}, nil
		}
		fakeWriter.SendSystemNotificationFn = func(ctx context.Context, notification dto.SystemNotification) error {
			notifications = append(notifications, notification)
			return nil
		}
		fakeCleaning.CleanRoomFn = func(ctx context.Context, request domain.CleaningOrder) error {
			return nil
		}

		breakfastCh := make(chan interface{}, 1)
		dinnerCh := make(chan interface{}, 1)
		checkoutCh := make(chan []domain.Booking, 1)
		fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
			switch hour {
			case 6:
				timerCh <- <-breakfastCh
			case 18:
				timerCh <- <-dinnerCh
			default:
				t.Fatalf("HourNotification() unexpected hour: %d", hour)
			}
		}
		fakeNotifications.CheckOutNotificationFn = func(ctx context.Context, bookingCh chan []domain.Booking) {
			bookingCh <- <-checkoutCh
		}

		breakfastCh <- struct{}{}
		dinnerCh <- struct{}{}
		dayDeliveries <- amqp.Delivery{
			Acknowledger: &mqfakes.FakeAcknowledger{},
			DeliveryTag:  1,
			Body:         mustMarshalTimeEventStay(t, time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)),
		}
		checkoutCh <- []domain.Booking{booking}

		ctx, cancel := context.WithCancel(context.Background())
		err = handler.Handle(ctx, booking)
		cancel()
		if err != nil {
			t.Fatalf("Handle() unexpected error: %v", err)
		}

		wantWaitedActions := []dto.GuestAction{
			dto.GuestAction_GO_FOR_DINNER,
			dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION,
			dto.GuestAction_GO_FOR_DINNER,
		}
		if len(waitedActions) != len(wantWaitedActions) {
			t.Fatalf("Handle() waited actions len = %d, want %d (%+v)", len(waitedActions), len(wantWaitedActions), waitedActions)
		}
		for i := range wantWaitedActions {
			if waitedActions[i] != wantWaitedActions[i] {
				t.Fatalf("Handle() waited action[%d] = %v, want %v", i, waitedActions[i], wantWaitedActions[i])
			}
		}
		if ackedActions != 9 {
			t.Fatalf("Handle() acked actions = %d, want 9", ackedActions)
		}
		if fakeCleaning.CleanRoomCallCount != 3 {
			t.Fatalf("Handle() cleaning calls = %d, want 3", fakeCleaning.CleanRoomCallCount)
		}
		if !containsNotification(notifications, dto.SystemNotification_CHECK_OUT_TODAY) {
			t.Fatalf("Handle() notifications = %+v, missing %v", notifications, dto.SystemNotification_CHECK_OUT_TODAY)
		}
	})

	t.Run("returns context error when canceled before first day change", func(t *testing.T) {
		booking := mustBooking(t, "cottage-9")
		fakeCleaning := &appfakes.CleaningService{}
		fakeWriter := &chatfakes.Writer{}
		fakeReader := &chatfakes.Reader{}
		fakeNotifications := &fakeNotificationService{}
		timeEvents, _, stopTimeEvents := newTestTimeEventService(t)
		defer stopTimeEvents()

		handler := NewStayHandler(fakeNotifications, fakeCleaning, timeEvents, fakeWriter, fakeReader)

		fakeReader.AckGuestActionFn = func(ctx context.Context) error { return nil }
		ctx, cancel := context.WithCancel(context.Background())
		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			if action == dto.GuestAction_GO_FOR_DINNER {
				cancel()
			}
			return &dto.ChatMessage{
				Payload: &dto.ChatMessage_GuestAction{GuestAction: action},
			}, nil
		}
		fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
			<-ctx.Done()
		}
		fakeNotifications.CheckOutNotificationFn = func(ctx context.Context, bookingCh chan []domain.Booking) {
			<-ctx.Done()
		}

		err := handler.Handle(ctx, booking)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Handle() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestStayHandlerStayRoutine(t *testing.T) {
	t.Parallel()

	fakeCleaning := &appfakes.CleaningService{}
	fakeReader := &chatfakes.Reader{}
	handler := NewStayHandler(&fakeNotificationService{}, fakeCleaning, nil, &chatfakes.Writer{}, fakeReader)
	booking := mustBooking(t, "cottage-12")

	waitedActions := make([]dto.GuestAction, 0, 2)
	ackedActions := 0
	cleaningRequests := make([]domain.CleaningOrder, 0, 2)

	fakeReader.AckGuestActionFn = func(ctx context.Context) error {
		ackedActions++
		return nil
	}
	fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
		waitedActions = append(waitedActions, action)
		return &dto.ChatMessage{
			Payload: &dto.ChatMessage_GuestAction{GuestAction: action},
		}, nil
	}
	fakeCleaning.CleanRoomFn = func(ctx context.Context, request domain.CleaningOrder) error {
		cleaningRequests = append(cleaningRequests, request)
		return nil
	}

	err := handler.stayRoutine(context.Background(), booking)
	if err != nil {
		t.Fatalf("stayRoutine() unexpected error: %v", err)
	}

	if ackedActions != 5 {
		t.Fatalf("stayRoutine() acked actions = %d, want 5", ackedActions)
	}
	wantWaitedActions := []dto.GuestAction{
		dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION,
		dto.GuestAction_GO_FOR_DINNER,
	}
	for i := range wantWaitedActions {
		if waitedActions[i] != wantWaitedActions[i] {
			t.Fatalf("stayRoutine() waited action[%d] = %v, want %v", i, waitedActions[i], wantWaitedActions[i])
		}
	}
	if len(cleaningRequests) != 2 {
		t.Fatalf("stayRoutine() cleaning requests len = %d, want 2", len(cleaningRequests))
	}
	if cleaningRequests[0].RoomName != booking.CottageName || cleaningRequests[0].RequestType != enum.FullCleaning {
		t.Fatalf("stayRoutine() first cleaning request = %+v", cleaningRequests[0])
	}
	if cleaningRequests[1].RoomName != booking.CottageName || cleaningRequests[1].RequestType != enum.PrepareForSleep {
		t.Fatalf("stayRoutine() second cleaning request = %+v", cleaningRequests[1])
	}
}

func TestStayHandlerCheckInDayRoutine(t *testing.T) {
	t.Parallel()

	fakeCleaning := &appfakes.CleaningService{}
	fakeReader := &chatfakes.Reader{}
	handler := NewStayHandler(&fakeNotificationService{}, fakeCleaning, nil, &chatfakes.Writer{}, fakeReader)
	booking := mustBooking(t, "cottage-14")

	ackedActions := 0
	waitedActions := make([]dto.GuestAction, 0, 1)

	fakeReader.AckGuestActionFn = func(ctx context.Context) error {
		ackedActions++
		return nil
	}
	fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
		waitedActions = append(waitedActions, action)
		return &dto.ChatMessage{
			Payload: &dto.ChatMessage_GuestAction{GuestAction: action},
		}, nil
	}

	err := handler.checkInDayRoutine(context.Background(), booking)
	if err != nil {
		t.Fatalf("checkInDayRoutine() unexpected error: %v", err)
	}

	if ackedActions != 3 {
		t.Fatalf("checkInDayRoutine() acked actions = %d, want 3", ackedActions)
	}
	if len(waitedActions) != 1 || waitedActions[0] != dto.GuestAction_GO_FOR_DINNER {
		t.Fatalf("checkInDayRoutine() waited actions = %+v", waitedActions)
	}
	if fakeCleaning.CleanRoomCallCount != 1 {
		t.Fatalf("checkInDayRoutine() cleaning calls = %d, want 1", fakeCleaning.CleanRoomCallCount)
	}
	if fakeCleaning.LastCleaningRequest.RoomName != booking.CottageName || fakeCleaning.LastCleaningRequest.RequestType != enum.PrepareForSleep {
		t.Fatalf("checkInDayRoutine() cleaning request = %+v", fakeCleaning.LastCleaningRequest)
	}
}

func TestStayHandlerWaitDayChangeEvent(t *testing.T) {
	t.Parallel()

	handler := NewStayHandler(&fakeNotificationService{}, &appfakes.CleaningService{}, nil, &chatfakes.Writer{}, &chatfakes.Reader{})

	t.Run("returns event", func(t *testing.T) {
		events := make(chan time.Time, 1)
		want := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
		events <- want

		got, err := handler.waitDayChangeEvent(context.Background(), events)
		if err != nil {
			t.Fatalf("waitDayChangeEvent() unexpected error: %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("waitDayChangeEvent() time = %v, want %v", got, want)
		}
	})

	t.Run("returns context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := handler.waitDayChangeEvent(ctx, make(chan time.Time))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waitDayChangeEvent() error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("returns stream closed error", func(t *testing.T) {
		events := make(chan time.Time)
		close(events)

		_, err := handler.waitDayChangeEvent(context.Background(), events)
		if !errors.Is(err, errDayChangeStreamClosed) {
			t.Fatalf("waitDayChangeEvent() error = %v, want %v", err, errDayChangeStreamClosed)
		}
	})
}

func TestStayHandlerNotifyBreakfast(t *testing.T) {
	t.Parallel()

	t.Run("sends breakfast notification when event arrives", func(t *testing.T) {
		fakeWriter := &chatfakes.Writer{}
		fakeNotifications := &fakeNotificationService{}
		handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, fakeWriter, &chatfakes.Reader{})

		events := make(chan interface{}, 1)
		ctx, cancel := context.WithCancel(context.Background())
		fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
			if hour != 6 {
				t.Fatalf("HourNotification() hour = %d, want 6", hour)
			}
			timerCh <- <-events
		}
		fakeWriter.SendSystemNotificationFn = func(ctx context.Context, notification dto.SystemNotification) error {
			cancel()
			return nil
		}

		events <- struct{}{}

		err := handler.notifyBreakfast(ctx)
		if err != nil {
			t.Fatalf("notifyBreakfast() unexpected error: %v", err)
		}
		if fakeWriter.SendSystemNotificationCallCount != 1 {
			t.Fatalf("notifyBreakfast() notification calls = %d, want 1", fakeWriter.SendSystemNotificationCallCount)
		}
		if fakeWriter.LastSendSystemNotificationValue != dto.SystemNotification_BREAKFAST_READY {
			t.Fatalf("notifyBreakfast() notification = %v, want %v", fakeWriter.LastSendSystemNotificationValue, dto.SystemNotification_BREAKFAST_READY)
		}
	})

	t.Run("wraps writer error", func(t *testing.T) {
		fakeWriter := &chatfakes.Writer{}
		fakeNotifications := &fakeNotificationService{}
		handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, fakeWriter, &chatfakes.Reader{})
		wantErr := errors.New("writer failed")

		events := make(chan interface{}, 1)
		fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
			timerCh <- <-events
		}
		fakeWriter.SendSystemNotificationFn = func(ctx context.Context, notification dto.SystemNotification) error {
			return wantErr
		}

		events <- struct{}{}

		err := handler.notifyBreakfast(context.Background())
		if err == nil || !errors.Is(err, wantErr) {
			t.Fatalf("notifyBreakfast() error = %v, want wrapped %v", err, wantErr)
		}
	})
}

func TestStayHandlerNotifyDinner(t *testing.T) {
	t.Parallel()

	fakeWriter := &chatfakes.Writer{}
	fakeNotifications := &fakeNotificationService{}
	handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, fakeWriter, &chatfakes.Reader{})

	events := make(chan interface{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
		if hour != 18 {
			t.Fatalf("HourNotification() hour = %d, want 18", hour)
		}
		timerCh <- <-events
	}
	fakeWriter.SendSystemNotificationFn = func(ctx context.Context, notification dto.SystemNotification) error {
		cancel()
		return nil
	}

	events <- struct{}{}

	err := handler.notifyDinner(ctx)
	if err != nil {
		t.Fatalf("notifyDinner() unexpected error: %v", err)
	}
	if fakeWriter.SendSystemNotificationCallCount != 1 {
		t.Fatalf("notifyDinner() notification calls = %d, want 1", fakeWriter.SendSystemNotificationCallCount)
	}
	if fakeWriter.LastSendSystemNotificationValue != dto.SystemNotification_DINNER_READY {
		t.Fatalf("notifyDinner() notification = %v, want %v", fakeWriter.LastSendSystemNotificationValue, dto.SystemNotification_DINNER_READY)
	}
}

func TestStayHandlerNotifyCheckoutToday(t *testing.T) {
	t.Parallel()

	t.Run("sends checkout notification when bookings exist", func(t *testing.T) {
		fakeWriter := &chatfakes.Writer{}
		fakeNotifications := &fakeNotificationService{}
		handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, fakeWriter, &chatfakes.Reader{})

		checkoutCh := make(chan []domain.Booking, 1)
		fakeNotifications.CheckOutNotificationFn = func(ctx context.Context, bookingCh chan []domain.Booking) {
			bookingCh <- <-checkoutCh
		}
		checkoutCh <- []domain.Booking{mustBooking(t, "cottage-20")}

		err := handler.notifyCheckoutToday(context.Background())
		if err != nil {
			t.Fatalf("notifyCheckoutToday() unexpected error: %v", err)
		}
		if fakeWriter.SendSystemNotificationCallCount != 1 {
			t.Fatalf("notifyCheckoutToday() notification calls = %d, want 1", fakeWriter.SendSystemNotificationCallCount)
		}
		if fakeWriter.LastSendSystemNotificationValue != dto.SystemNotification_CHECK_OUT_TODAY {
			t.Fatalf("notifyCheckoutToday() notification = %v, want %v", fakeWriter.LastSendSystemNotificationValue, dto.SystemNotification_CHECK_OUT_TODAY)
		}
	})

	t.Run("returns context error when canceled before notification arrives", func(t *testing.T) {
		fakeWriter := &chatfakes.Writer{}
		fakeNotifications := &fakeNotificationService{}
		handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, fakeWriter, &chatfakes.Reader{})

		ctx, cancel := context.WithCancel(context.Background())
		fakeNotifications.CheckOutNotificationFn = func(ctx context.Context, bookingCh chan []domain.Booking) {
			<-ctx.Done()
		}

		cancel()

		err := handler.notifyCheckoutToday(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("notifyCheckoutToday() error = %v, want %v", err, context.Canceled)
		}
		if fakeWriter.SendSystemNotificationCallCount != 0 {
			t.Fatalf("notifyCheckoutToday() notification calls = %d, want 0", fakeWriter.SendSystemNotificationCallCount)
		}
	})
}

type fakeNotificationService struct {
	HourNotificationFn     func(ctx context.Context, timerCh chan interface{}, hour int)
	CheckOutNotificationFn func(ctx context.Context, bookingCh chan []domain.Booking)
}

func (f *fakeNotificationService) HourNotification(ctx context.Context, timerCh chan interface{}, hour int) {
	if f.HourNotificationFn != nil {
		f.HourNotificationFn(ctx, timerCh, hour)
		return
	}
}

func (f *fakeNotificationService) CheckOutNotification(ctx context.Context, bookingCh chan []domain.Booking) {
	if f.CheckOutNotificationFn != nil {
		f.CheckOutNotificationFn(ctx, bookingCh)
		return
	}
}

func newTestTimeEventService(t *testing.T) (app.TimeEventService, chan amqp.Delivery, func()) {
	t.Helper()

	hourConsumer := &mqfakes.FakeMqConsumer{
		ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
			ch := make(chan amqp.Delivery)
			go func() {
				<-ctx.Done()
				close(ch)
			}()
			return ch, nil
		},
	}
	dayDeliveries := make(chan amqp.Delivery, 2)
	dayConsumer := &mqfakes.FakeMqConsumer{
		ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
			return dayDeliveries, nil
		},
	}

	service, err := app.NewTimeEventService(hourConsumer, dayConsumer)
	if err != nil {
		t.Fatalf("NewTimeEventService() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.Start(ctx)
		close(done)
	}()

	return service, dayDeliveries, func() {
		cancel()
		close(dayDeliveries)
		<-done
	}
}

func mustMarshalTimeEventStay(t *testing.T, eventTime time.Time) []byte {
	t.Helper()

	payload, err := proto.Marshal(&dto.TimeEvent{
		Time: timestamppb.New(eventTime),
	})
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	return payload
}

func containsNotification(notifications []dto.SystemNotification, want dto.SystemNotification) bool {
	for _, notification := range notifications {
		if notification == want {
			return true
		}
	}

	return false
}
