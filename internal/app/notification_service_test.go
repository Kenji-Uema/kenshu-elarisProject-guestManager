package app

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/infra/mq/fakes"
)

func TestNotificationServiceHourNotification(t *testing.T) {
	t.Parallel()

	service, timeEvents := newTestNotificationService(t)
	notifications := make(chan interface{}, 1)
	go service.HourNotification(context.Background(), notifications, 6)

	waitForCondition(t, time.Second, func() bool {
		timeEvents.hourChangeChannelsMu.RLock()
		defer timeEvents.hourChangeChannelsMu.RUnlock()
		return len(timeEvents.hourChangeChannels.Values()) == 1
	})

	want := time.Date(2026, 3, 23, 6, 0, 0, 0, time.UTC)
	timeEvents.notifyHourChange(context.Background(), want)

	select {
	case got, ok := <-notifications:
		if !ok {
			t.Fatal("HourNotification() channel closed unexpectedly")
		}
		gotTime, ok := got.(time.Time)
		if !ok {
			t.Fatalf("HourNotification() payload type = %T, want time.Time", got)
		}
		if !gotTime.Equal(want) {
			t.Fatalf("HourNotification() payload = %v, want %v", gotTime, want)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for hour notification")
	}

	timeEvents.notifyHourChange(context.Background(), want.Add(time.Hour))
	select {
	case got := <-notifications:
		t.Fatalf("HourNotification() received unexpected payload for non-target hour: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestNotificationServiceHourNotificationUnregistersOnCancel(t *testing.T) {
	t.Parallel()

	service, timeEvents := newTestNotificationService(t)
	ctx, cancel := context.WithCancel(context.Background())
	notifications := make(chan interface{}, 1)
	done := make(chan struct{})
	go func() {
		service.HourNotification(ctx, notifications, 6)
		close(done)
	}()

	waitForCondition(t, time.Second, func() bool {
		timeEvents.hourChangeChannelsMu.RLock()
		defer timeEvents.hourChangeChannelsMu.RUnlock()
		return len(timeEvents.hourChangeChannels.Values()) == 1
	})

	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for HourNotification() to return after cancel")
	}

	waitForCondition(t, time.Second, func() bool {
		timeEvents.hourChangeChannelsMu.RLock()
		defer timeEvents.hourChangeChannelsMu.RUnlock()
		return len(timeEvents.hourChangeChannels.Values()) == 0
	})
}

func TestNotificationServiceCheckOutNotificationUnregistersOnCancel(t *testing.T) {
	t.Parallel()

	service, timeEvents := newTestNotificationService(t)
	ctx, cancel := context.WithCancel(context.Background())
	notifications := make(chan []domain.Booking, 1)
	done := make(chan struct{})
	go func() {
		service.CheckOutNotification(ctx, notifications)
		close(done)
	}()

	waitForCondition(t, time.Second, func() bool {
		timeEvents.dayChangeChannelsMu.RLock()
		defer timeEvents.dayChangeChannelsMu.RUnlock()
		return len(timeEvents.dayChangeChannels.Values()) == 1
	})

	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for CheckOutNotification() to return after cancel")
	}

	waitForCondition(t, time.Second, func() bool {
		timeEvents.dayChangeChannelsMu.RLock()
		defer timeEvents.dayChangeChannelsMu.RUnlock()
		return len(timeEvents.dayChangeChannels.Values()) == 0
	})
}

func newTestNotificationService(t *testing.T) (notificationService, *timeEventService) {
	t.Helper()

	timeEvents, err := NewTimeEventService(&fakes.FakeMqConsumer{}, &fakes.FakeMqConsumer{})
	if err != nil {
		t.Fatalf("NewTimeEventService() error = %v", err)
	}

	concrete, ok := timeEvents.(*timeEventService)
	if !ok {
		t.Fatalf("NewTimeEventService() returned %T, want *timeEventService", timeEvents)
	}

	return notificationService{
		timeEventService: concrete,
	}, concrete
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition not met before timeout")
}
