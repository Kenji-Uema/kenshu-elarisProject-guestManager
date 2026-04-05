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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var receptionServiceTracer = otel.Tracer("guest-manager.app.reception-service")

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
	ctx, span := receptionServiceTracer.Start(ctx, "ReceptionService.CheckIn")
	defer span.End()
	span.SetAttributes(
		attribute.String("guest.document_id", document),
		attribute.String("booking.checkin_date", today.UTC().Format(time.RFC3339)),
	)

	guest, err := r.guestService.GetByDocument(ctx, document)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_lookup_failed")
		return domain.Booking{}, fmt.Errorf("failed to get guest %s: %w", document, err)
	}

	booking, err := r.getFromCache(ctx, today, err, guest)
	if err != nil || booking == (domain.Booking{}) {
		booking, err = r.guestService.GetBookingByDate(ctx, document, today)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "booking_lookup_failed")
			return domain.Booking{}, fmt.Errorf("failed to get booking for guest %s: %w", document, err)
		}
	}

	if booking == (domain.Booking{}) {
		err := fmt.Errorf("no booking found for guest %s", document)
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_not_found")
		return domain.Booking{}, err
	}

	span.SetAttributes(
		attribute.String("booking.id", booking.Id.Hex()),
		attribute.String("booking.cottage_name", booking.CottageName),
	)

	if err := r.processCheckIn(ctx, document, booking, guest, today); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "checkin_processing_failed")
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
	ctx, span := receptionServiceTracer.Start(ctx, "ReceptionService.CheckOut")
	defer span.End()
	span.SetAttributes(
		attribute.String("booking.id", booking.Id.Hex()),
		attribute.String("booking.cottage_name", booking.CottageName),
		attribute.String("booking.checkout_date", today.UTC().Format(time.RFC3339)),
	)

	// if !sameUTCDay(booking.StayPeriod.CheckOut, today) {
	// 	err := fmt.Errorf("checkout day not today")
	// 	span.RecordError(err)
	// 	span.SetStatus(codes.Error, "checkout_day_mismatch")
	// 	return err
	// }

	cottageName := booking.CottageName
	if err := r.bookingRepo.UpdateStatus(ctx, booking.Id, enum.BookingStatusPast); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_status_update_failed")
		return fmt.Errorf("failed to update booking status for room %s: %w", cottageName, err)
	}

	if err := r.cottageRepo.RemovePastBooking(ctx, booking.Id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove_past_booking_failed")
		return fmt.Errorf("failed to remove past booking for room %s: %w", cottageName, err)
	}

	if err := r.cottageRepo.UpdateCurrentGuest(ctx, cottageName, bson.NilObjectID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "current_guest_clear_failed")
		return fmt.Errorf("failed to update current guest for room %s: %w", cottageName, err)
	}

	cleaningRequest, err := domain.NewCleaningOrder(cottageName, enum.FullCleaning)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cleaning_order_creation_failed")
		return fmt.Errorf("failed to create cleaning request for room %s: %w", cottageName, err)
	}
	if err := r.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cleaning_request_failed")
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
	ctx, span := receptionServiceTracer.Start(ctx, "ReceptionService.getFromCache")
	defer span.End()

	var booking domain.Booking
	redisKey := fmt.Sprintf("checkin.%s", today.UTC().Format("2006-01-02"))
	span.SetAttributes(
		attribute.String("cache.key", redisKey),
		attribute.String("guest.id", guest.Id.Hex()),
	)
	cachedBookingsJSON, err := r.cache.GetBytes(ctx, redisKey)

	if err != nil {
		span.SetAttributes(attribute.Bool("cache.hit", false))
		return domain.Booking{}, nil
	}
	span.SetAttributes(attribute.Bool("cache.hit", true))

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
	ctx, span := receptionServiceTracer.Start(ctx, "ReceptionService.processCheckIn")
	defer span.End()
	span.SetAttributes(
		attribute.String("guest.document_id", document),
		attribute.String("guest.id", guest.Id.Hex()),
		attribute.String("booking.id", booking.Id.Hex()),
		attribute.String("booking.cottage_name", booking.CottageName),
	)

	if !sameUTCDay(booking.StayPeriod.CheckIn, today) {
		err := fmt.Errorf("checkin day not today")
		span.RecordError(err)
		span.SetStatus(codes.Error, "checkin_day_mismatch")
		return err
	}

	if booking.Status != enum.BookingStatusConfirmed {
		err := fmt.Errorf("booking for guest %s is not confirmed", document)
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_not_confirmed")
		return err
	}

	cottage, err := r.cottageRepo.GetByName(ctx, booking.CottageName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cottage_lookup_failed")
		return fmt.Errorf("failed to get cottage %s: %w", booking.CottageName, err)
	}

	if cottage.CurrentGuest != bson.NilObjectID || cottage.CleaningStatus != enum.FullyCleaned {
		err := fmt.Errorf("cottage %s is not available", booking.CottageName)
		span.RecordError(err)
		span.SetStatus(codes.Error, "cottage_not_ready")
		return err
	}

	if err := r.cottageRepo.UpdateCurrentGuest(ctx, booking.CottageName, guest.Id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "current_guest_update_failed")
		return fmt.Errorf("failed to update current guest for room %s: %w", booking.CottageName, err)
	}

	cleaningRequest, err := domain.NewCleaningOrder(booking.CottageName, enum.PrepareForGuest)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cleaning_order_creation_failed")
		return fmt.Errorf("failed to create cleaning request for room %s: %w", booking.CottageName, err)
	}
	if err := r.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cleaning_request_failed")
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
