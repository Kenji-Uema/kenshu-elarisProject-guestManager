package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	appfakes "github.com/Kenji-Uema/guestManager/internal/app/fakes"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	mdbfakes "github.com/Kenji-Uema/guestManager/internal/infra/mdb/fakes"
	mqfakes "github.com/Kenji-Uema/guestManager/internal/infra/mq/fakes"
	"github.com/Kenji-Uema/guestManager/internal/port"
	portfakes "github.com/Kenji-Uema/guestManager/internal/port/fakes"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestReceptionServiceCheckIn(t *testing.T) {
	t.Run("returns error when booking checkin date is not today", func(t *testing.T) {
		guest := mustGuestForReception(t)
		booking := mustReceptionBooking(
			t,
			time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
			enum.BookingStatusConfirmed,
		)
		guestService := &appfakes.GuestService{
			GetByDocumentFn: func(ctx context.Context, documentId string) (domain.Guest, error) {
				return guest, nil
			},
			GetBookingByDateFn: func(ctx context.Context, document string, date time.Time) (domain.Booking, error) {
				return booking, nil
			},
		}
		cleaningService := &appfakes.CleaningService{}
		cottageRepo := &mdbfakes.FakeCottageRepo{}
		cache := &portfakes.FakeCache{
			GetBytesFn: func(ctx context.Context, key string) ([]byte, error) {
				return nil, port.ErrCacheMiss
			},
		}

		service, err := app.NewReceptionService(
			guestService,
			cleaningService,
			cottageRepo,
			&mdbfakes.FakeBookingRepo{},
			&mqfakes.FakeMqConsumer{},
			cache,
		)
		if err != nil {
			t.Fatalf("NewReceptionService() error = %v", err)
		}

		_, err = service.CheckIn(context.Background(), "doc-123", time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC))
		if err == nil || err.Error() != "checkin day not today" {
			t.Fatalf("CheckIn() error = %v, want %q", err, "checkin day not today")
		}
		if cleaningService.CleanRoomCallCount != 0 {
			t.Fatalf("CleanRoom() call count = %d, want 0", cleaningService.CleanRoomCallCount)
		}
		if cottageRepo.UpdateCurrentGuestCallCount != 0 {
			t.Fatalf("UpdateCurrentGuest() call count = %d, want 0", cottageRepo.UpdateCurrentGuestCallCount)
		}
	})

	t.Run("prepares cottage for guest on successful checkin", func(t *testing.T) {
		guest := mustGuestForReception(t)
		today := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)
		booking := mustReceptionBooking(
			t,
			today,
			time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
			enum.BookingStatusConfirmed,
		)
		guestService := &appfakes.GuestService{
			GetByDocumentFn: func(ctx context.Context, documentId string) (domain.Guest, error) {
				return guest, nil
			},
			GetBookingByDateFn: func(ctx context.Context, document string, date time.Time) (domain.Booking, error) {
				return booking, nil
			},
		}
		cleaningService := &appfakes.CleaningService{}
		cottageRepo := &mdbfakes.FakeCottageRepo{
			GetByNameFn: func(ctx context.Context, roomName string) (documents.Cottage, error) {
				return documents.Cottage{
					Name:           roomName,
					CurrentGuest:   bson.NilObjectID,
					CleaningStatus: enum.FullyCleaned,
				}, nil
			},
		}
		cache := &portfakes.FakeCache{
			GetBytesFn: func(ctx context.Context, key string) ([]byte, error) {
				return nil, port.ErrCacheMiss
			},
		}

		service, err := app.NewReceptionService(
			guestService,
			cleaningService,
			cottageRepo,
			&mdbfakes.FakeBookingRepo{},
			&mqfakes.FakeMqConsumer{},
			cache,
		)
		if err != nil {
			t.Fatalf("NewReceptionService() error = %v", err)
		}

		got, err := service.CheckIn(context.Background(), "doc-123", today)
		if err != nil {
			t.Fatalf("CheckIn() error = %v", err)
		}
		if got.CottageName != booking.CottageName {
			t.Fatalf("CheckIn() booking = %+v, want cottage %q", got, booking.CottageName)
		}
		if cottageRepo.UpdateCurrentGuestCallCount != 1 {
			t.Fatalf("UpdateCurrentGuest() call count = %d, want 1", cottageRepo.UpdateCurrentGuestCallCount)
		}
		if cottageRepo.LastUpdateCurrentGuestRoom != booking.CottageName {
			t.Fatalf("UpdateCurrentGuest() room = %q, want %q", cottageRepo.LastUpdateCurrentGuestRoom, booking.CottageName)
		}
		if cottageRepo.LastUpdateCurrentGuestId != guest.Id {
			t.Fatalf("UpdateCurrentGuest() guest id = %s, want %s", cottageRepo.LastUpdateCurrentGuestId.Hex(), guest.Id.Hex())
		}
		if cleaningService.CleanRoomCallCount != 1 {
			t.Fatalf("CleanRoom() call count = %d, want 1", cleaningService.CleanRoomCallCount)
		}
		if cleaningService.LastCleaningRequest.RoomName != booking.CottageName {
			t.Fatalf("CleanRoom() room = %q, want %q", cleaningService.LastCleaningRequest.RoomName, booking.CottageName)
		}
		if cleaningService.LastCleaningRequest.RequestType != enum.PrepareForGuest {
			t.Fatalf("CleanRoom() request type = %v, want %v", cleaningService.LastCleaningRequest.RequestType, enum.PrepareForGuest)
		}
	})
}

func TestReceptionServiceCheckOut(t *testing.T) {
	t.Run("removes booking even when checkout date is not today", func(t *testing.T) {
		booking := mustReceptionBooking(
			t,
			time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
			enum.BookingStatusConfirmed,
		)
		cleaningService := &appfakes.CleaningService{}
		cottageRepo := &mdbfakes.FakeCottageRepo{}
		bookingRepo := &mdbfakes.FakeBookingRepo{}

		service, err := app.NewReceptionService(
			&appfakes.GuestService{},
			cleaningService,
			cottageRepo,
			bookingRepo,
			&mqfakes.FakeMqConsumer{},
			&portfakes.FakeCache{},
		)
		if err != nil {
			t.Fatalf("NewReceptionService() error = %v", err)
		}

		err = service.CheckOut(context.Background(), booking, time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("CheckOut() error = %v", err)
		}
		if bookingRepo.UpdateStatusCallCount != 1 {
			t.Fatalf("UpdateStatus() call count = %d, want 1", bookingRepo.UpdateStatusCallCount)
		}
		if cottageRepo.RemovePastBookingCallCount != 1 {
			t.Fatalf("RemovePastBooking() call count = %d, want 1", cottageRepo.RemovePastBookingCallCount)
		}
		if cleaningService.CleanRoomCallCount != 1 {
			t.Fatalf("CleanRoom() call count = %d, want 1", cleaningService.CleanRoomCallCount)
		}
	})

	t.Run("removes booking, clears current guest, and requests full cleaning", func(t *testing.T) {
		today := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		booking := mustReceptionBooking(
			t,
			time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			today,
			enum.BookingStatusConfirmed,
		)
		cleaningService := &appfakes.CleaningService{}
		cottageRepo := &mdbfakes.FakeCottageRepo{}
		bookingRepo := &mdbfakes.FakeBookingRepo{}

		service, err := app.NewReceptionService(
			&appfakes.GuestService{},
			cleaningService,
			cottageRepo,
			bookingRepo,
			&mqfakes.FakeMqConsumer{},
			&portfakes.FakeCache{},
		)
		if err != nil {
			t.Fatalf("NewReceptionService() error = %v", err)
		}

		err = service.CheckOut(context.Background(), booking, today)
		if err != nil {
			t.Fatalf("CheckOut() error = %v", err)
		}
		if bookingRepo.UpdateStatusCallCount != 1 {
			t.Fatalf("UpdateStatus() call count = %d, want 1", bookingRepo.UpdateStatusCallCount)
		}
		if bookingRepo.LastUpdateStatusBookingId != booking.Id {
			t.Fatalf("UpdateStatus() booking id = %s, want %s", bookingRepo.LastUpdateStatusBookingId.Hex(), booking.Id.Hex())
		}
		if bookingRepo.LastUpdateStatus != enum.BookingStatusPast {
			t.Fatalf("UpdateStatus() status = %v, want %v", bookingRepo.LastUpdateStatus, enum.BookingStatusPast)
		}
		if cottageRepo.RemovePastBookingCallCount != 1 {
			t.Fatalf("RemovePastBooking() call count = %d, want 1", cottageRepo.RemovePastBookingCallCount)
		}
		if cottageRepo.LastRemovePastBookingId != booking.Id {
			t.Fatalf("RemovePastBooking() booking id = %s, want %s", cottageRepo.LastRemovePastBookingId.Hex(), booking.Id.Hex())
		}
		if cottageRepo.UpdateCurrentGuestCallCount != 1 {
			t.Fatalf("UpdateCurrentGuest() call count = %d, want 1", cottageRepo.UpdateCurrentGuestCallCount)
		}
		if cottageRepo.LastUpdateCurrentGuestRoom != booking.CottageName {
			t.Fatalf("UpdateCurrentGuest() room = %q, want %q", cottageRepo.LastUpdateCurrentGuestRoom, booking.CottageName)
		}
		if cottageRepo.LastUpdateCurrentGuestId != bson.NilObjectID {
			t.Fatalf("UpdateCurrentGuest() guest id = %s, want nil object id", cottageRepo.LastUpdateCurrentGuestId.Hex())
		}
		if cleaningService.CleanRoomCallCount != 1 {
			t.Fatalf("CleanRoom() call count = %d, want 1", cleaningService.CleanRoomCallCount)
		}
		if cleaningService.LastCleaningRequest.RoomName != booking.CottageName {
			t.Fatalf("CleanRoom() room = %q, want %q", cleaningService.LastCleaningRequest.RoomName, booking.CottageName)
		}
		if cleaningService.LastCleaningRequest.RequestType != enum.FullCleaning {
			t.Fatalf("CleanRoom() request type = %v, want %v", cleaningService.LastCleaningRequest.RequestType, enum.FullCleaning)
		}
	})
}

func TestReceptionServiceCheckinFallback(t *testing.T) {
	t.Run("returns error when fallback booking is not valid for today", func(t *testing.T) {
		guest := mustGuestForReception(t)
		bookingRepo := &mdbfakes.FakeBookingRepo{
			FindByGuestIdAndBookingNumberFn: func(ctx context.Context, guestId bson.ObjectID, bookingNumber string) (documents.Booking, error) {
				return documents.Booking{
					Id:             bson.NewObjectID(),
					MainGuest:      guest.Id,
					NumberOfGuests: 2,
					StayPeriod: documents.Period{
						CheckIn:  time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
						CheckOut: time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
					},
					CottageName: "cottage-7",
					Status:      enum.BookingStatusConfirmed,
				}, nil
			},
		}
		guestService := &appfakes.GuestService{
			GetByDocumentFn: func(ctx context.Context, documentId string) (domain.Guest, error) {
				return guest, nil
			},
		}
		cleaningService := &appfakes.CleaningService{}

		service, err := app.NewReceptionService(
			guestService,
			cleaningService,
			&mdbfakes.FakeCottageRepo{},
			bookingRepo,
			&mqfakes.FakeMqConsumer{},
			&portfakes.FakeCache{},
		)
		if err != nil {
			t.Fatalf("NewReceptionService() error = %v", err)
		}

		_, err = service.CheckinFallback(context.Background(), "doc-123", "booking-42", time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC))
		if err == nil || err.Error() != "booking booking-42 is not valid for check-in on 2026-03-24" {
			t.Fatalf("CheckinFallback() error = %v, want booking date validation error", err)
		}
		if cleaningService.CleanRoomCallCount != 0 {
			t.Fatalf("CleanRoom() call count = %d, want 0", cleaningService.CleanRoomCallCount)
		}
	})

	t.Run("prepares cottage for guest on successful fallback checkin", func(t *testing.T) {
		guest := mustGuestForReception(t)
		today := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)
		bookingRepo := &mdbfakes.FakeBookingRepo{
			FindByGuestIdAndBookingNumberFn: func(ctx context.Context, guestId bson.ObjectID, bookingNumber string) (documents.Booking, error) {
				return documents.Booking{
					Id:             bson.NewObjectID(),
					MainGuest:      guest.Id,
					NumberOfGuests: 2,
					StayPeriod: documents.Period{
						CheckIn:  today,
						CheckOut: time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
					},
					CottageName: "cottage-7",
					Status:      enum.BookingStatusConfirmed,
				}, nil
			},
		}
		guestService := &appfakes.GuestService{
			GetByDocumentFn: func(ctx context.Context, documentId string) (domain.Guest, error) {
				return guest, nil
			},
		}
		cleaningService := &appfakes.CleaningService{}
		cottageRepo := &mdbfakes.FakeCottageRepo{
			GetByNameFn: func(ctx context.Context, roomName string) (documents.Cottage, error) {
				return documents.Cottage{
					Name:           roomName,
					CurrentGuest:   bson.NilObjectID,
					CleaningStatus: enum.FullyCleaned,
				}, nil
			},
		}

		service, err := app.NewReceptionService(
			guestService,
			cleaningService,
			cottageRepo,
			bookingRepo,
			&mqfakes.FakeMqConsumer{},
			&portfakes.FakeCache{},
		)
		if err != nil {
			t.Fatalf("NewReceptionService() error = %v", err)
		}

		got, err := service.CheckinFallback(context.Background(), "doc-123", "booking-42", today)
		if err != nil {
			t.Fatalf("CheckinFallback() error = %v", err)
		}
		if got.CottageName != "cottage-7" {
			t.Fatalf("CheckinFallback() booking = %+v, want cottage-7", got)
		}
		if cottageRepo.UpdateCurrentGuestCallCount != 1 {
			t.Fatalf("UpdateCurrentGuest() call count = %d, want 1", cottageRepo.UpdateCurrentGuestCallCount)
		}
		if cleaningService.CleanRoomCallCount != 1 {
			t.Fatalf("CleanRoom() call count = %d, want 1", cleaningService.CleanRoomCallCount)
		}
		if cleaningService.LastCleaningRequest.RequestType != enum.PrepareForGuest {
			t.Fatalf("CleanRoom() request type = %v, want %v", cleaningService.LastCleaningRequest.RequestType, enum.PrepareForGuest)
		}
	})
}

func TestReceptionServiceReceiveCottageKey(t *testing.T) {
	cottageRepo := &mdbfakes.FakeCottageRepo{}

	service, err := app.NewReceptionService(
		&appfakes.GuestService{},
		&appfakes.CleaningService{},
		cottageRepo,
		&mdbfakes.FakeBookingRepo{},
		&mqfakes.FakeMqConsumer{},
		&portfakes.FakeCache{},
	)
	if err != nil {
		t.Fatalf("NewReceptionService() error = %v", err)
	}

	err = service.ReceiveCottageKey(context.Background(), "cottage-7", "key-7")
	if err != nil {
		t.Fatalf("ReceiveCottageKey() error = %v", err)
	}
	if cottageRepo.UpdateKeyHolderCallCount != 1 {
		t.Fatalf("UpdateKeyHolder() call count = %d, want 1", cottageRepo.UpdateKeyHolderCallCount)
	}
	if cottageRepo.LastUpdateKeyHolderRoom != "cottage-7" {
		t.Fatalf("UpdateKeyHolder() room = %q, want %q", cottageRepo.LastUpdateKeyHolderRoom, "cottage-7")
	}
	if cottageRepo.LastUpdateKeyNumber != "key-7" {
		t.Fatalf("UpdateKeyHolder() key number = %q, want %q", cottageRepo.LastUpdateKeyNumber, "key-7")
	}
	if cottageRepo.LastUpdateKeyHolder != enum.KeyHolderGuest {
		t.Fatalf("UpdateKeyHolder() holder = %v, want %v", cottageRepo.LastUpdateKeyHolder, enum.KeyHolderGuest)
	}
}

func TestReceptionServiceReturnCottageKey(t *testing.T) {
	cottageRepo := &mdbfakes.FakeCottageRepo{}

	service, err := app.NewReceptionService(
		&appfakes.GuestService{},
		&appfakes.CleaningService{},
		cottageRepo,
		&mdbfakes.FakeBookingRepo{},
		&mqfakes.FakeMqConsumer{},
		&portfakes.FakeCache{},
	)
	if err != nil {
		t.Fatalf("NewReceptionService() error = %v", err)
	}

	err = service.ReturnCottageKey(context.Background(), "cottage-7", "key-7")
	if err != nil {
		t.Fatalf("ReturnCottageKey() error = %v", err)
	}
	if cottageRepo.UpdateKeyHolderCallCount != 1 {
		t.Fatalf("UpdateKeyHolder() call count = %d, want 1", cottageRepo.UpdateKeyHolderCallCount)
	}
	if cottageRepo.LastUpdateKeyHolderRoom != "cottage-7" {
		t.Fatalf("UpdateKeyHolder() room = %q, want %q", cottageRepo.LastUpdateKeyHolderRoom, "cottage-7")
	}
	if cottageRepo.LastUpdateKeyNumber != "key-7" {
		t.Fatalf("UpdateKeyHolder() key number = %q, want %q", cottageRepo.LastUpdateKeyNumber, "key-7")
	}
	if cottageRepo.LastUpdateKeyHolder != enum.KeyHolderCottage {
		t.Fatalf("UpdateKeyHolder() holder = %v, want %v", cottageRepo.LastUpdateKeyHolder, enum.KeyHolderCottage)
	}
}

func mustGuestForReception(t *testing.T) domain.Guest {
	t.Helper()

	now := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	guest, err := domain.NewGuest(
		bson.NewObjectID(),
		"doc-123",
		"John",
		"Doe",
		"john@example.com",
		"Billing Street 1",
		&now,
		&now,
	)
	if err != nil {
		t.Fatalf("NewGuest() error = %v", err)
	}
	return guest
}

func mustReceptionBooking(t *testing.T, checkIn time.Time, checkOut time.Time, status enum.BookingStatus) domain.Booking {
	t.Helper()

	stayPeriod, err := domain.NewPeriod(checkIn, checkOut)
	if err != nil {
		t.Fatalf("NewPeriod() error = %v", err)
	}

	booking, err := domain.NewBooking(
		bson.NewObjectID(),
		bson.NewObjectID(),
		2,
		stayPeriod,
		"cottage-7",
		status,
	)
	if err != nil {
		t.Fatalf("NewBooking() error = %v", err)
	}
	return booking
}
