package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/app/fakes"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	mdbfakes "github.com/Kenji-Uema/guestManager/internal/infra/mdb/fakes"
	portfakes "github.com/Kenji-Uema/guestManager/internal/port/fakes"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestArrangeCheckinConsumesDayChangeEvent(t *testing.T) {
	cachedKeys := make(chan string, 2)
	registered := make(chan struct{})
	timeEvents := &fakes.TimeEventService{
		RegisterFn: func(eventType app.TimeEventType, ch chan<- time.Time) {
			close(registered)
		},
	}

	checkInDate := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	checkOutDate := time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC)
	bookingRepo := &mdbfakes.FakeBookingRepo{
		FindByCheckInDateFn: func(ctx context.Context, date time.Time) ([]documents.Booking, error) {
			return []documents.Booking{
				{
					Id:             bson.NewObjectID(),
					MainGuest:      bson.NewObjectID(),
					NumberOfGuests: 2,
					StayPeriod: documents.Period{
						CheckIn:  checkInDate,
						CheckOut: checkOutDate,
					},
					CottageName: "Cottage 1",
					Status:      "confirmed",
				},
			}, nil
		},
	}
	cache := &portfakes.FakeCache{}
	cache.SetBytesFn = func(ctx context.Context, key string, value []byte, expiration time.Duration) error {
		cachedKeys <- key
		return nil
	}

	service, err := app.NewArrangeCottageService(bookingRepo, timeEvents, cache)
	if err != nil {
		t.Fatalf("NewArrangeCottageService() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		service.ArrangeCheckIn(ctx)
	}()

	select {
	case <-registered:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for day change registration")
	}

	select {
	case timeEvents.LastRegisterChannel <- time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC):
	case <-time.After(time.Second):
		t.Fatal("timed out sending day change event")
	}

	gotKeys := make([]string, 0, 2)
	var firstKey string
	select {
	case firstKey = <-cachedKeys:
	case <-time.After(time.Second):
		t.Fatal("expected first cache write after day change event")
	}
	gotKeys = append(gotKeys, firstKey)

	var secondKey string
	select {
	case secondKey = <-cachedKeys:
	case <-time.After(time.Second):
		t.Fatal("expected second cache write after day change event")
	}
	gotKeys = append(gotKeys, secondKey)

	if !bookingRepo.LastFindByCheckInDate.Equal(checkInDate) {
		t.Fatalf("FindByCheckInDate() date = %v, want %v", bookingRepo.LastFindByCheckInDate, checkInDate)
	}
	if gotKeys[0] != "checkin.2026-03-25" {
		t.Fatalf("first cache key = %q, want %q", gotKeys[0], "checkin.2026-03-25")
	}
	if gotKeys[1] != "checkout.2026-03-27" {
		t.Fatalf("second cache key = %q, want %q", gotKeys[1], "checkout.2026-03-27")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ArrangeCheckIn() did not return after context cancellation")
	}
}
