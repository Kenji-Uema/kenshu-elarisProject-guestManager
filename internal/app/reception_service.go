package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/infra/redis"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ReceptionService interface {
	CheckIn(ctx context.Context, guestDocument string, today time.Time) error
	CheckOut(ctx context.Context, guestDocument string, today time.Time) error
	CheckinFallback(ctx context.Context, document string, bookingNumber string, today time.Time) error
}

type receptionService struct {
	guestService    GuestService
	cleaningService CleaningService

	cottageRepo port.CottageRepo
	bookingRepo port.BookingRepo

	clockEventConsumer port.MqConsumer

	redis *redis.Redis
}

func NewReceptionService(guestService GuestService, cleaningService CleaningService,
	cottageRepo port.CottageRepo, bookingRepo port.BookingRepo,
	clockEventConsumer port.MqConsumer, redisClient *redis.Redis) (ReceptionService, error) {
	if err := validation.New().
		NotZeroValue("guestService", guestService).
		NotZeroValue("cleaningService", cleaningService).
		NotZeroValue("cottageRepo", cottageRepo).
		NotZeroValue("bookingRepo", bookingRepo).
		NotZeroValue("clockEventConsumer", clockEventConsumer).
		NotZeroValue("redisClient", redisClient).
		Validate(); err != nil {
		return nil, fmt.Errorf("NewReceptionService: %w", err)
	}

	return &receptionService{
		guestService:       guestService,
		cleaningService:    cleaningService,
		cottageRepo:        cottageRepo,
		bookingRepo:        bookingRepo,
		clockEventConsumer: clockEventConsumer,
		redis:              redisClient,
	}, nil
}

func (r receptionService) CheckIn(ctx context.Context, document string, today time.Time) error {
	guest, err := r.guestService.GetByDocument(ctx, document)
	if err != nil {
		return fmt.Errorf("failed to get guest %s: %w", document, err)
	}

	booking, err := r.getFromCache(ctx, today, err, guest)
	if err != nil || booking == (domain.Booking{}) {
		booking, err = r.guestService.GetBookingByDate(ctx, document, today)
		if err != nil {
			return fmt.Errorf("failed to get booking for guest %s: %w", document, err)
		}
	}

	if booking == (domain.Booking{}) {
		return fmt.Errorf("no booking found for guest %s", document)
	}

	if err := r.processCheckIn(ctx, document, booking, guest); err != nil {
		return err
	}
	return nil
}

func (r receptionService) CheckinFallback(ctx context.Context, document string, bookingNumber string, today time.Time) error {
	guest, err := r.guestService.GetByDocument(ctx, document)
	if err != nil {
		return fmt.Errorf("failed to get guest %s: %w", document, err)
	}

	bookingDoc, err := r.bookingRepo.FindByGuestIdAndBookingNumber(ctx, guest.Id, bookingNumber)
	if err != nil {
		return fmt.Errorf("failed to get booking for guest %s: %w", document, err)
	}
	if bookingDoc == (documents.Booking{}) {
		return fmt.Errorf("no booking found for guest %s", document)
	}

	startOfDay := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	if !startOfDay(bookingDoc.StayPeriod.Start).Equal(startOfDay(today)) {
		return fmt.Errorf("booking %s is not valid for check-in on %s", bookingNumber, today.UTC().Format("2006-01-02"))
	}

	stayPeriod, err := domain.NewPeriod(bookingDoc.StayPeriod.Start, bookingDoc.StayPeriod.End)
	if err != nil {
		return fmt.Errorf("failed to map booking period for guest %s: %w", document, err)
	}

	booking, err := domain.NewBooking(
		bookingDoc.MainGuest,
		bookingDoc.NumberOfGuests,
		stayPeriod,
		bookingDoc.CottageName,
		bookingDoc.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to map booking for guest %s: %w", document, err)
	}

	if err := r.processCheckIn(ctx, document, booking, guest); err != nil {
		return err
	}
	return nil
}

func (r receptionService) CheckOut(ctx context.Context, document string, today time.Time) error {
	guest, err := r.guestService.GetByDocument(ctx, document)
	if err != nil {
		return fmt.Errorf("failed to get guest %s: %w", document, err)
	}

	allBookings, err := r.bookingRepo.FindByGuestId(ctx, guest.Id)
	if err != nil {
		return fmt.Errorf("failed to find bookings for guest %s: %w", document, err)
	}

	startOfDay := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}

	todayStart := startOfDay(today)
	bookingIdx := -1
	for i := range allBookings {
		if startOfDay(allBookings[i].StayPeriod.End).Equal(todayStart) {
			bookingIdx = i
			break
		}
	}

	if bookingIdx == -1 {
		return fmt.Errorf("no booking ending today for guest %s", document)
	}

	cottageName := allBookings[bookingIdx].CottageName
	if err := r.cottageRepo.ClearCurrentGuest(ctx, cottageName); err != nil {
		return fmt.Errorf("failed to clear current guest for room %s: %w", cottageName, err)
	}

	cleaningRequest, err := domain.NewCleaningRequest(cottageName, "CLEAN")
	if err != nil {
		return fmt.Errorf("failed to create cleaning request for room %s: %w", cottageName, err)
	}
	if err := r.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return fmt.Errorf("failed to request cleaning for room %s: %w", cottageName, err)
	}

	return nil
}

func (r receptionService) getFromCache(ctx context.Context, today time.Time, err error, guest domain.Guest) (domain.Booking, error) {
	var booking domain.Booking
	redisKey := fmt.Sprintf("checkin.%s", today.UTC().Format("2006-01-02"))
	cachedBookingsJSON, err := r.redis.Client().Get(ctx, redisKey).Bytes()

	if err != nil {
		return domain.Booking{}, nil
	}

	var cachedBookings []documents.Booking
	if err := json.Unmarshal(cachedBookingsJSON, &cachedBookings); err == nil {
		for _, b := range cachedBookings {
			if b.MainGuest != guest.Id {
				continue
			}

			stayPeriod, periodErr := domain.NewPeriod(b.StayPeriod.Start, b.StayPeriod.End)
			if periodErr != nil {
				break
			}

			booking, err = domain.NewBooking(
				b.MainGuest,
				b.NumberOfGuests,
				stayPeriod,
				b.CottageName,
				b.Status,
			)
			if err == nil {
				break
			}
		}
	}

	return booking, nil
}

func (r receptionService) processCheckIn(ctx context.Context, document string, booking domain.Booking, guest domain.Guest) error {
	if !r.BookingOk(booking) {
		return fmt.Errorf("booking for guest %s is not confirmed", document)
	}

	if !r.CottageOk(ctx, booking.CottageName) {
		return fmt.Errorf("cottage %s is not available", booking.CottageName)
	}

	if err := r.cottageRepo.UpdateCurrentGuest(ctx, booking.CottageName, guest.Id); err != nil {
		return fmt.Errorf("failed to update current guest for room %s: %w", booking.CottageName, err)
	}
	return nil
}

func (r receptionService) BookingOk(booking domain.Booking) bool {
	return booking.Status == "confirmed"
}

func (r receptionService) CottageOk(ctx context.Context, cottageName string) bool {
	if err := validation.New().NotBlank("cottageName", cottageName).Validate(); err != nil {
		return false
	}

	cottage, err := r.cottageRepo.GetByName(ctx, cottageName)
	if err != nil {
		return false
	}

	isNotOccupied := cottage.CurrentGuest == bson.NilObjectID
	isCleaned := cottage.Cleaned

	return isNotOccupied && isCleaned
}
