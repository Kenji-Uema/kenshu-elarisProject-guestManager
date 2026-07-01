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
	infrafakes "github.com/Kenji-Uema/guestManager/internal/infra/clock/fakes"
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

		handler := NewStayHandler(fakeNotifications, fakeCleaning, timeEvents, nil, fakeWriter, fakeReader)

		waitedActions := make([]dto.GuestAction, 0, 9)
		notifications := make([]dto.SystemNotification, 0, 3)
		secondDayEventSent := false

		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			waitedActions = append(waitedActions, action)
			if action == dto.GuestAction_GO_FOR_DINNER && len(waitedActions) == 10 && !secondDayEventSent {
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

		fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
			switch hour {
			case 6, 18:
				timerCh <- struct{}{}
			default:
				t.Fatalf("HourNotification() unexpected hour: %d", hour)
			}
		}
		dayDeliveries <- amqp.Delivery{
			Acknowledger: &mqfakes.FakeAcknowledger{},
			DeliveryTag:  1,
			Body:         mustMarshalTimeEventStay(t, time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)),
		}
		ctx, cancel := context.WithCancel(context.Background())
		err = handler.Handle(ctx, booking)
		cancel()
		if err != nil {
			t.Fatalf("Handle() unexpected error: %v", err)
		}

		wantWaitedActions := []dto.GuestAction{
			dto.GuestAction_ENTER_COTTAGE,
			dto.GuestAction_GO_FOR_A_BATH,
			dto.GuestAction_GO_FOR_DINNER,
			dto.GuestAction_GO_TO_SLEEP,
			dto.GuestAction_WAKEUP,
			dto.GuestAction_GO_FOR_BREAKFAST,
			dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION,
			dto.GuestAction_ENJOY_RESORT,
			dto.GuestAction_GO_FOR_A_BATH,
			dto.GuestAction_GO_FOR_DINNER,
			dto.GuestAction_GO_TO_SLEEP,
			dto.GuestAction_WAKEUP,
			dto.GuestAction_LEAVE_COTTAGE,
		}
		if len(waitedActions) != len(wantWaitedActions) {
			t.Fatalf("Handle() waited actions len = %d, want %d (%+v)", len(waitedActions), len(wantWaitedActions), waitedActions)
		}
		for i := range wantWaitedActions {
			if waitedActions[i] != wantWaitedActions[i] {
				t.Fatalf("Handle() waited action[%d] = %v, want %v", i, waitedActions[i], wantWaitedActions[i])
			}
		}
		if fakeCleaning.CleanRoomCallCount != 3 {
			t.Fatalf("Handle() cleaning calls = %d, want 3", fakeCleaning.CleanRoomCallCount)
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

		handler := NewStayHandler(fakeNotifications, fakeCleaning, timeEvents, nil, fakeWriter, fakeReader)

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
			select {
			case timerCh <- struct{}{}:
			case <-ctx.Done():
			}
		}
		err := handler.Handle(ctx, booking)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Handle() error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("skips stay routine when first day change is already checkout day", func(t *testing.T) {
		booking := mustBooking(t, "cottage-10")
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

		handler := NewStayHandler(fakeNotifications, fakeCleaning, timeEvents, nil, fakeWriter, fakeReader)

		waitedActions := make([]dto.GuestAction, 0, 5)
		notifications := make([]dto.SystemNotification, 0, 2)

		fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
			waitedActions = append(waitedActions, action)
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

		fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
			switch hour {
			case 6, 18:
				timerCh <- struct{}{}
			default:
				t.Fatalf("HourNotification() unexpected hour: %d", hour)
			}
		}
		dayDeliveries <- amqp.Delivery{
			Acknowledger: &mqfakes.FakeAcknowledger{},
			DeliveryTag:  1,
			Body:         mustMarshalTimeEventStay(t, time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)),
		}

		err = handler.Handle(context.Background(), booking)
		if err != nil {
			t.Fatalf("Handle() unexpected error: %v", err)
		}

		wantWaitedActions := []dto.GuestAction{
			dto.GuestAction_ENTER_COTTAGE,
			dto.GuestAction_GO_FOR_A_BATH,
			dto.GuestAction_GO_FOR_DINNER,
			dto.GuestAction_GO_TO_SLEEP,
			dto.GuestAction_WAKEUP,
			dto.GuestAction_LEAVE_COTTAGE,
		}
		if len(waitedActions) != len(wantWaitedActions) {
			t.Fatalf("Handle() waited actions len = %d, want %d (%+v)", len(waitedActions), len(wantWaitedActions), waitedActions)
		}
		for i := range wantWaitedActions {
			if waitedActions[i] != wantWaitedActions[i] {
				t.Fatalf("Handle() waited action[%d] = %v, want %v", i, waitedActions[i], wantWaitedActions[i])
			}
		}
		if fakeCleaning.CleanRoomCallCount != 1 {
			t.Fatalf("Handle() cleaning calls = %d, want 1", fakeCleaning.CleanRoomCallCount)
		}
	})

}

func TestStayHandlerWaitForStayTransitionUsesClockFallback(t *testing.T) {
	t.Parallel()

	booking := mustBooking(t, "cottage-11")
	stayPeriod, err := domain.NewPeriod(
		time.Date(2026, 3, 22, 15, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewPeriod() error = %v", err)
	}
	booking.StayPeriod = stayPeriod

	currentTime := time.Date(2026, 3, 23, 23, 0, 0, 0, time.UTC)
	fakeClock := &infrafakes.FakeClockClient{
		NowFn: func(ctx context.Context) (*time.Time, error) {
			now := currentTime
			return &now, nil
		},
	}

	handler := NewStayHandler(&fakeNotificationService{}, &appfakes.CleaningService{}, nil, fakeClock, &chatfakes.Writer{}, &chatfakes.Reader{})
	events := make(chan time.Time, 1)
	events <- time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	checkoutReached, err := handler.waitForStayTransition(context.Background(), booking, events)
	if err != nil {
		t.Fatalf("waitForStayTransition() unexpected error = %v", err)
	}
	if checkoutReached {
		t.Fatalf("waitForStayTransition() checkoutReached = true, want false before checkout day")
	}

	currentTime = time.Date(2026, 3, 24, 0, 1, 0, 0, time.UTC)

	checkoutReached, err = handler.waitForStayTransition(context.Background(), booking, events)
	if err != nil {
		t.Fatalf("waitForStayTransition() unexpected error after clock advance = %v", err)
	}
	if !checkoutReached {
		t.Fatalf("waitForStayTransition() checkoutReached = false, want true after checkout day")
	}
}

func TestStayHandlerStayRoutine(t *testing.T) {
	t.Parallel()

	fakeCleaning := &appfakes.CleaningService{}
	fakeReader := &chatfakes.Reader{}
	fakeNotifications := &fakeNotificationService{}
	fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
		timerCh <- struct{}{}
	}
	handler := NewStayHandler(fakeNotifications, fakeCleaning, nil, nil, &chatfakes.Writer{}, fakeReader)
	booking := mustBooking(t, "cottage-12")

	waitedActions := make([]dto.GuestAction, 0, 5)
	cleaningRequests := make([]domain.CleaningOrder, 0, 2)

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

	wantWaitedActions := []dto.GuestAction{
		dto.GuestAction_WAKEUP,
		dto.GuestAction_GO_FOR_BREAKFAST,
		dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION,
		dto.GuestAction_ENJOY_RESORT,
		dto.GuestAction_GO_FOR_A_BATH,
		dto.GuestAction_GO_FOR_DINNER,
		dto.GuestAction_GO_TO_SLEEP,
	}
	if len(waitedActions) != len(wantWaitedActions) {
		t.Fatalf("stayRoutine() waited actions len = %d, want %d (%+v)", len(waitedActions), len(wantWaitedActions), waitedActions)
	}
	for i := range wantWaitedActions {
		if waitedActions[i] != wantWaitedActions[i] {
			t.Fatalf("stayRoutine() waited action[%d] = %v, want %v", i, waitedActions[i], wantWaitedActions[i])
		}
	}
	if len(cleaningRequests) != 2 {
		t.Fatalf("stayRoutine() cleaning requests len = %d, want 2", len(cleaningRequests))
	}
	if cleaningRequests[0].RoomName != booking.CottageName || cleaningRequests[0].RequestType != enum.DailyCleaning {
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
	fakeNotifications := &fakeNotificationService{}
	fakeNotifications.HourNotificationFn = func(ctx context.Context, timerCh chan interface{}, hour int) {
		timerCh <- struct{}{}
	}
	handler := NewStayHandler(fakeNotifications, fakeCleaning, nil, nil, &chatfakes.Writer{}, fakeReader)
	booking := mustBooking(t, "cottage-14")

	waitedActions := make([]dto.GuestAction, 0, 4)
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

	wantWaitedActions := []dto.GuestAction{
		dto.GuestAction_ENTER_COTTAGE,
		dto.GuestAction_GO_FOR_A_BATH,
		dto.GuestAction_GO_FOR_DINNER,
		dto.GuestAction_GO_TO_SLEEP,
	}
	if len(waitedActions) != len(wantWaitedActions) {
		t.Fatalf("checkInDayRoutine() waited actions len = %d, want %d (%+v)", len(waitedActions), len(wantWaitedActions), waitedActions)
	}
	for i := range wantWaitedActions {
		if waitedActions[i] != wantWaitedActions[i] {
			t.Fatalf("checkInDayRoutine() waited action[%d] = %v, want %v", i, waitedActions[i], wantWaitedActions[i])
		}
	}
	if len(waitedActions) != 4 {
		t.Fatalf("checkInDayRoutine() waited actions = %+v", waitedActions)
	}
	if fakeCleaning.CleanRoomCallCount != 1 {
		t.Fatalf("checkInDayRoutine() cleaning calls = %d, want 1", fakeCleaning.CleanRoomCallCount)
	}
	if fakeCleaning.LastCleaningRequest.RoomName != booking.CottageName || fakeCleaning.LastCleaningRequest.RequestType != enum.PrepareForSleep {
		t.Fatalf("checkInDayRoutine() cleaning request = %+v", fakeCleaning.LastCleaningRequest)
	}
}

func TestStayHandlerCheckoutDayRoutine(t *testing.T) {
	t.Parallel()

	fakeReader := &chatfakes.Reader{}
	handler := NewStayHandler(&fakeNotificationService{}, &appfakes.CleaningService{}, nil, nil, &chatfakes.Writer{}, fakeReader)

	waitedActions := make([]dto.GuestAction, 0, 2)
	fakeReader.WaitForGuestActionFn = func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
		waitedActions = append(waitedActions, action)
		return &dto.ChatMessage{
			Payload: &dto.ChatMessage_GuestAction{GuestAction: action},
		}, nil
	}

	err := handler.checkoutDayRoutine(context.Background())
	if err != nil {
		t.Fatalf("checkoutDayRoutine() unexpected error: %v", err)
	}

	wantWaitedActions := []dto.GuestAction{
		dto.GuestAction_WAKEUP,
		dto.GuestAction_LEAVE_COTTAGE,
	}
	if len(waitedActions) != len(wantWaitedActions) {
		t.Fatalf("checkoutDayRoutine() waited actions len = %d, want %d (%+v)", len(waitedActions), len(wantWaitedActions), waitedActions)
	}
	for i := range wantWaitedActions {
		if waitedActions[i] != wantWaitedActions[i] {
			t.Fatalf("checkoutDayRoutine() waited action[%d] = %v, want %v", i, waitedActions[i], wantWaitedActions[i])
		}
	}
}

func TestStayHandlerWaitDayChangeEvent(t *testing.T) {
	t.Parallel()

	handler := NewStayHandler(&fakeNotificationService{}, &appfakes.CleaningService{}, nil, nil, &chatfakes.Writer{}, &chatfakes.Reader{})

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

func TestIsCheckoutToday(t *testing.T) {
	t.Parallel()

	booking := mustBooking(t, "cottage-18")
	stayPeriod, err := domain.NewPeriod(
		time.Date(2026, 3, 22, 15, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewPeriod() error = %v", err)
	}
	booking.StayPeriod = stayPeriod

	tests := []struct {
		name      string
		timeEvent time.Time
		want      bool
	}{
		{
			name:      "before checkout day",
			timeEvent: time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
			want:      false,
		},
		{
			name:      "on checkout day",
			timeEvent: time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name:      "after checkout day",
			timeEvent: time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCheckoutToday(booking, tt.timeEvent)
			if got != tt.want {
				t.Fatalf("isCheckoutToday() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStayHandlerNotifyBreakfast(t *testing.T) {
	t.Parallel()

	t.Run("sends breakfast notification when event arrives", func(t *testing.T) {
		fakeWriter := &chatfakes.Writer{}
		fakeNotifications := &fakeNotificationService{}
		handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, nil, fakeWriter, &chatfakes.Reader{})

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
		handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, nil, fakeWriter, &chatfakes.Reader{})
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
	handler := NewStayHandler(fakeNotifications, &appfakes.CleaningService{}, nil, nil, fakeWriter, &chatfakes.Reader{})

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
