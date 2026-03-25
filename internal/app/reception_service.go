package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ReceptionService interface {
	CheckIn(ctx context.Context, guestDocument string, today time.Time) (domain.Booking, error)
	CheckOut(ctx context.Context, booking domain.Booking, today time.Time) error
	CheckinFallback(ctx context.Context, document string, bookingNumber string, today time.Time) (domain.Booking, error)
	ReceiveCottageKey(ctx context.Context, cottageName string, keyNumber string) error
	ReturnCottageKey(ctx context.Context, cottageName string, keyNumber string) error
}

type receptionService struct {
	guestService       GuestService
	cleaningService    CleaningService
	cottageRepo        port.CottageRepo
	bookingRepo        port.BookingRepo
	clockEventConsumer port.MqConsumer
	cache              port.Cache
}

func NewReceptionService(guestService GuestService, cleaningService CleaningService,
	cottageRepo port.CottageRepo, bookingRepo port.BookingRepo,
	clockEventConsumer port.MqConsumer, cache port.Cache) (ReceptionService, error) {
	if err := validation.New().
		NotZeroValue("guestService", guestService).
		NotZeroValue("cleaningService", cleaningService).
		NotZeroValue("cottageRepo", cottageRepo).
		NotZeroValue("bookingRepo", bookingRepo).
		NotZeroValue("clockEventConsumer", clockEventConsumer).
		NotZeroValue("cache", cache).
		Validate(); err != nil {
		return nil, fmt.Errorf("NewReceptionService: %w", err)
	}

	return &receptionService{
		guestService:       guestService,
		cleaningService:    cleaningService,
		cottageRepo:        cottageRepo,
		bookingRepo:        bookingRepo,
		clockEventConsumer: clockEventConsumer,
		cache:              cache,
	}, nil
}

func (r receptionService) CheckIn(ctx context.Context, document string, today time.Time) (domain.Booking, error) {
	guest, err := r.guestService.GetByDocument(ctx, document)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("failed to get guest %s: %w", document, err)
	}

	booking, err := r.getFromCache(ctx, today, err, guest)
	if err != nil || booking == (domain.Booking{}) {
		booking, err = r.guestService.GetBookingByDate(ctx, document, today)
		if err != nil {
			return domain.Booking{}, fmt.Errorf("failed to get booking for guest %s: %w", document, err)
		}
	}

	if booking == (domain.Booking{}) {
		return domain.Booking{}, fmt.Errorf("no booking found for guest %s", document)
	}

	if err := r.processCheckIn(ctx, document, booking, guest, today); err != nil {
		return domain.Booking{}, err
	}
	return booking, nil
}

func (r receptionService) CheckinFallback(ctx context.Context, document string, bookingNumber string, today time.Time) (domain.Booking, error) {
	guest, err := r.guestService.GetByDocument(ctx, document)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("failed to get guest %s: %w", document, err)
	}

	bookingDoc, err := r.bookingRepo.FindByGuestIdAndBookingNumber(ctx, guest.Id, bookingNumber)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("failed to get booking for guest %s: %w", document, err)
	}
	if bookingDoc == (documents.Booking{}) {
		return domain.Booking{}, fmt.Errorf("no booking found for guest %s", document)
	}

	startOfDay := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	if !startOfDay(bookingDoc.StayPeriod.CheckIn).Equal(startOfDay(today)) {
		return domain.Booking{}, fmt.Errorf("booking %s is not valid for check-in on %s", bookingNumber, today.UTC().Format("2006-01-02"))
	}

	stayPeriod, err := domain.NewPeriod(bookingDoc.StayPeriod.CheckIn, bookingDoc.StayPeriod.CheckOut)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("failed to map booking period for guest %s: %w", document, err)
	}

	booking, err := domain.NewBooking(
		bookingDoc.Id,
		bookingDoc.MainGuest,
		bookingDoc.NumberOfGuests,
		stayPeriod,
		bookingDoc.CottageName,
		bookingDoc.Status,
	)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("failed to map booking for guest %s: %w", document, err)
	}

	if err := r.processCheckIn(ctx, document, booking, guest, today); err != nil {
		return domain.Booking{}, err
	}
	return booking, nil
}

func (r receptionService) CheckOut(ctx context.Context, booking domain.Booking, today time.Time) error {
	if !sameUTCDay(booking.StayPeriod.CheckOut, today) {
		return fmt.Errorf("checkout day not today")
	}

	cottageName := booking.CottageName
	if err := r.bookingRepo.UpdateStatus(ctx, booking.Id, enum.BookingStatusPast); err != nil {
		return fmt.Errorf("failed to update booking status for room %s: %w", cottageName, err)
	}

	if err := r.cottageRepo.RemovePastBooking(ctx, booking.Id); err != nil {
		return fmt.Errorf("failed to remove past booking for room %s: %w", cottageName, err)
	}

	if err := r.cottageRepo.UpdateCurrentGuest(ctx, cottageName, bson.NilObjectID); err != nil {
		return fmt.Errorf("failed to update current guest for room %s: %w", cottageName, err)
	}

	cleaningRequest, err := domain.NewCleaningOrder(cottageName, enum.FullCleaning)
	if err != nil {
		return fmt.Errorf("failed to create cleaning request for room %s: %w", cottageName, err)
	}
	if err := r.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return fmt.Errorf("failed to request cleaning for room %s: %w", cottageName, err)
	}

	return nil
}

func (r receptionService) ReceiveCottageKey(ctx context.Context, cottageName string, keyNumber string) error {
	if err := r.cottageRepo.UpdateKeyHolder(ctx, cottageName, keyNumber, enum.KeyHolderGuest); err != nil {
		return fmt.Errorf("failed to update cottage key holder for room %s: %w", cottageName, err)
	}

	return nil
}

func (r receptionService) ReturnCottageKey(ctx context.Context, cottageName string, keyNumber string) error {
	if err := r.cottageRepo.UpdateKeyHolder(ctx, cottageName, keyNumber, enum.KeyHolderCottage); err != nil {
		return fmt.Errorf("failed to update cottage key holder for room %s: %w", cottageName, err)
	}

	return nil
}

func (r receptionService) getFromCache(ctx context.Context, today time.Time, err error, guest domain.Guest) (domain.Booking, error) {
	var booking domain.Booking
	redisKey := fmt.Sprintf("checkin.%s", today.UTC().Format("2006-01-02"))
	cachedBookingsJSON, err := r.cache.GetBytes(ctx, redisKey)

	if err != nil {
		return domain.Booking{}, nil
	}

	var cachedBookings []domain.Booking
	if err := json.Unmarshal(cachedBookingsJSON, &cachedBookings); err == nil {
		for _, b := range cachedBookings {
			if b.MainGuest != guest.Id {
				continue
			}
			booking = b
			break
		}
	}

	return booking, nil
}

func (r receptionService) processCheckIn(ctx context.Context, document string, booking domain.Booking, guest domain.Guest, today time.Time) error {
	if !sameUTCDay(booking.StayPeriod.CheckIn, today) {
		return fmt.Errorf("checkin day not today")
	}

	if booking.Status != enum.BookingStatusConfirmed {
		return fmt.Errorf("booking for guest %s is not confirmed", document)
	}

	cottage, err := r.cottageRepo.GetByName(ctx, booking.CottageName)
	if err != nil {
		return fmt.Errorf("failed to get cottage %s: %w", booking.CottageName, err)
	}

	if cottage.CurrentGuest != bson.NilObjectID || cottage.CleaningStatus != enum.FullyCleaned {
		return fmt.Errorf("cottage %s is not available", booking.CottageName)
	}

	if err := r.cottageRepo.UpdateCurrentGuest(ctx, booking.CottageName, guest.Id); err != nil {
		return fmt.Errorf("failed to update current guest for room %s: %w", booking.CottageName, err)
	}

	cleaningRequest, err := domain.NewCleaningOrder(booking.CottageName, enum.PrepareForGuest)
	if err != nil {
		return fmt.Errorf("failed to create cleaning request for room %s: %w", booking.CottageName, err)
	}
	if err := r.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return fmt.Errorf("failed to request cottage preparation for room %s: %w", booking.CottageName, err)
	}

	return nil
}

func sameUTCDay(left time.Time, right time.Time) bool {
	left = left.UTC()
	right = right.UTC()

	return left.Year() == right.Year() &&
		left.Month() == right.Month() &&
		left.Day() == right.Day()
}
