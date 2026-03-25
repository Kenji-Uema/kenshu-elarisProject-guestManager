package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
)

type ArrangeCottageService interface {
	ArrangeCheckIn(ctx context.Context)
}

type arrangeCottageService struct {
	bookingRepo      port.BookingRepo
	timeEventService TimeEventService
	cache            port.Cache
}

func NewArrangeCottageService(bookingRepo port.BookingRepo, timeEventService TimeEventService, cache port.Cache) (ArrangeCottageService, error) {
	if err := validation.New().
		NotZeroValue("bookingRepo", bookingRepo).
		NotZeroValue("timeEventService", timeEventService).
		NotZeroValue("cache", cache).Validate(); err != nil {

		return nil, fmt.Errorf("NewArrangeCottageService: %w", err)
	}

	return &arrangeCottageService{
		bookingRepo:      bookingRepo,
		timeEventService: timeEventService,
		cache:            cache,
	}, nil
}

func (s arrangeCottageService) ArrangeCheckIn(ctx context.Context) {
	events := make(chan time.Time, 1)
	s.timeEventService.Register(TimeEventDayChange, events)
	defer s.timeEventService.Unregister(TimeEventDayChange, events)

	for {
		select {
		case <-ctx.Done():
			return
		case today, ok := <-events:
			if !ok {
				slog.WarnContext(ctx, "day change event stream closed")
				return
			}

			tomorrow := today.AddDate(0, 0, 1)

			bookingDocs, err := s.bookingRepo.FindByCheckInDate(ctx, tomorrow)
			if err != nil {
				slog.ErrorContext(ctx, "failed to find bookings by check-in date", "date", tomorrow, "error", err)
				continue
			}

			bookings := s.bookingsDocToDomain(ctx, bookingDocs)
			slog.InfoContext(ctx, "parsed bookings for day-change event", "count", len(bookings), "checkInDate", tomorrow)

			if err := s.cacheCheckIn(ctx, err, bookings, tomorrow); err != nil {
				slog.ErrorContext(ctx, "failed to cache check-in bookings", "error", err)
				continue
			}
			if err := s.cacheCheckOut(ctx, bookings); err != nil {
				slog.ErrorContext(ctx, "failed to cache check-out bookings", "error", err)
				continue
			}

			slog.InfoContext(ctx, "coming bookings load into redis")
		}
	}
}

func (s arrangeCottageService) bookingsDocToDomain(ctx context.Context, bookingDocs []documents.Booking) []domain.Booking {
	bookings := make([]domain.Booking, 0, len(bookingDocs))
	for _, b := range bookingDocs {
		stayPeriod, err := domain.NewPeriod(b.StayPeriod.CheckIn, b.StayPeriod.CheckOut)
		if err != nil {
			slog.ErrorContext(ctx, "failed to map booking stay period", "bookingId", b.Id.Hex(), "error", err)
			continue
		}

		booking, err := domain.NewBooking(
			b.Id,
			b.MainGuest,
			b.NumberOfGuests,
			stayPeriod,
			b.CottageName,
			b.Status,
		)
		if err != nil {
			var validationErr *validationErrors.ErrValidationConstrain
			if errors.As(err, &validationErr) {
				slog.WarnContext(ctx, "invalid booking data for day-change event",
					"bookingId", b.Id.Hex(), "field", validationErr.Field, "reason", validationErr.Message)
				continue
			}

			slog.ErrorContext(ctx, "failed to map booking to domain", "bookingId", b.Id.Hex(), "error", err)
			continue
		}

		bookings = append(bookings, booking)
	}
	return bookings
}

func (s arrangeCottageService) cacheCheckIn(ctx context.Context, err error, bookings []domain.Booking, tomorrow time.Time) error {
	var payload []byte
	if payload, err = json.Marshal(bookings); err != nil {
		slog.ErrorContext(ctx, "failed to marshal check-in bookings", "checkInDate", tomorrow, "error", err)
		return err
	}

	redisKey := fmt.Sprintf("checkin.%s", tomorrow.UTC().Format("2006-01-02"))
	if err := s.cache.SetBytes(ctx, redisKey, payload, 30*time.Minute); err != nil {
		slog.ErrorContext(ctx, "failed to save check-in bookings in redis", "key", redisKey, "error", err)
		return err
	}
	return nil
}

func (s arrangeCottageService) cacheCheckOut(ctx context.Context, bookings []domain.Booking) error {
	bookingsByCheckOutDate := make(map[string][]domain.Booking)
	for _, booking := range bookings {
		checkOutDate := booking.StayPeriod.CheckOut.UTC().Format("2006-01-02")
		bookingsByCheckOutDate[checkOutDate] = append(bookingsByCheckOutDate[checkOutDate], booking)
	}

	var cacheCheckoutErr error
	for checkOutDate, checkOutBookings := range bookingsByCheckOutDate {
		payload, err := json.Marshal(checkOutBookings)
		if err != nil {
			slog.ErrorContext(ctx, "failed to marshal check-out bookings", "checkOutDate", checkOutDate, "error", err)
			cacheCheckoutErr = errors.Join(cacheCheckoutErr, err)
		}

		checkOutKey := fmt.Sprintf("checkout.%s", checkOutDate)
		if err := s.cache.SetBytes(ctx, checkOutKey, payload, 30*time.Minute); err != nil {
			slog.ErrorContext(ctx, "failed to save check-out bookings in redis", "key", checkOutKey, "error", err)
			cacheCheckoutErr = errors.Join(cacheCheckoutErr, err)
		}
	}

	if cacheCheckoutErr != nil {
		return cacheCheckoutErr
	}
	return nil
}
