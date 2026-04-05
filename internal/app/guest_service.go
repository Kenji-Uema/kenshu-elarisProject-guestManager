package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var guestServiceTracer = otel.Tracer("guest-manager.app.guest-service")

type GuestService interface {
	GetById(ctx context.Context, id bson.ObjectID) (domain.Guest, error)
	GetByDocument(ctx context.Context, documentId string) (domain.Guest, error)
	Add(ctx context.Context, guest domain.Guest) (bson.ObjectID, error)
	Update(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error)
	GetBookings(ctx context.Context, guestId bson.ObjectID) ([]domain.Booking, error)
	GetBookingByDate(ctx context.Context, document string, date time.Time) (domain.Booking, error)
}

type guestService struct {
	guestRepo   port.GuestRepo
	bookingRepo port.BookingRepo
}

func NewGuestService(guestRepo port.GuestRepo, bookingRepo port.BookingRepo) (GuestService, error) {
	if err := validation.New().
		NotZeroValue("guestRepo", guestRepo).
		NotZeroValue("bookingRepo", bookingRepo).
		Validate(); err != nil {
		return nil, fmt.Errorf("NewGuestService: %w", err)
	}

	return &guestService{guestRepo: guestRepo, bookingRepo: bookingRepo}, nil
}

func (s *guestService) GetById(ctx context.Context, id bson.ObjectID) (domain.Guest, error) {
	ctx, span := guestServiceTracer.Start(ctx, "GuestService.GetById")
	defer span.End()
	span.SetAttributes(attribute.String("guest.id", id.Hex()))

	guestDoc, err := s.guestRepo.GetById(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_repo.get_by_id_failed")
		return domain.Guest{}, err
	}

	guest, err := domain.NewGuest(guestDoc.Id, guestDoc.DocumentId, guestDoc.GivenNames, guestDoc.Surname,
		guestDoc.Email, guestDoc.BillingAddress, guestDoc.CreatedAt, guestDoc.LastUpdate)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "guest_mapping_failed")
			return domain.Guest{}, err
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_mapping_failed")
		return domain.Guest{}, fmt.Errorf("guestService.GetById(%s): mapping to domain: %w", id.Hex(), err)
	}

	return guest, nil
}

func (s *guestService) GetByDocument(ctx context.Context, documentId string) (domain.Guest, error) {
	ctx, span := guestServiceTracer.Start(ctx, "GuestService.GetByDocument")
	defer span.End()
	span.SetAttributes(attribute.String("guest.document_id", documentId))

	guestDoc, err := s.guestRepo.GetByDocument(ctx, documentId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_repo.get_by_document_failed")
		return domain.Guest{}, err
	}

	guest, err := domain.NewGuest(guestDoc.Id, guestDoc.DocumentId, guestDoc.GivenNames, guestDoc.Surname,
		guestDoc.Email, guestDoc.BillingAddress, guestDoc.CreatedAt, guestDoc.LastUpdate)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "guest_mapping_failed")
			return domain.Guest{}, err
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_mapping_failed")
		return domain.Guest{}, fmt.Errorf("guestService.GetByDocument(%s): mapping to domain: %w", documentId, err)
	}

	return guest, nil

}

func (s *guestService) Add(ctx context.Context, guest domain.Guest) (bson.ObjectID, error) {
	ctx, span := guestServiceTracer.Start(ctx, "GuestService.Add")
	defer span.End()
	span.SetAttributes(attribute.String("guest.id", guest.Id.Hex()))

	id, err := s.guestRepo.Add(ctx, guest.ToMongoDoc())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_repo.add_failed")
		return bson.NilObjectID, err
	}

	span.SetAttributes(attribute.String("guest.created_id", id.Hex()))
	return id, nil
}

func (s *guestService) Update(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error) {
	ctx, span := guestServiceTracer.Start(ctx, "GuestService.Update")
	defer span.End()
	span.SetAttributes(attribute.String("guest.id", id.Hex()))

	updatedGuestDoc, err := s.guestRepo.Update(ctx, id, guest.ToMongoDoc())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_repo.update_failed")
		return domain.Guest{}, err
	}

	updatedGuest, err := domain.NewGuest(updatedGuestDoc.Id, updatedGuestDoc.DocumentId, updatedGuestDoc.GivenNames,
		updatedGuestDoc.Surname, updatedGuestDoc.Email, updatedGuestDoc.BillingAddress, updatedGuestDoc.CreatedAt, updatedGuestDoc.LastUpdate)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "guest_mapping_failed")
			return domain.Guest{}, err
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_mapping_failed")
		return domain.Guest{}, fmt.Errorf("guestService.Update(%s): mapping to domain: %w", id.Hex(), err)
	}

	return updatedGuest, nil
}

func (s *guestService) GetBookings(ctx context.Context, guestId bson.ObjectID) ([]domain.Booking, error) {
	ctx, span := guestServiceTracer.Start(ctx, "GuestService.GetBookings")
	defer span.End()
	span.SetAttributes(attribute.String("guest.id", guestId.Hex()))

	bookingsDoc, err := s.bookingRepo.FindByGuestId(ctx, guestId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_repo.find_by_guest_id_failed")
		return nil, err
	}

	if len(bookingsDoc) == 0 {
		span.SetAttributes(attribute.Int("guest.bookings.count", 0))
		return []domain.Booking{}, nil
	}

	bookings := make([]domain.Booking, 0, len(bookingsDoc))
	for _, b := range bookingsDoc {
		stayPeriod, err := domain.NewPeriod(b.StayPeriod.CheckIn, b.StayPeriod.CheckOut)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "booking_period_mapping_failed")
			return nil, err
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
			span.RecordError(err)
			span.SetStatus(codes.Error, "booking_mapping_failed")
			return nil, err
		}
		bookings = append(bookings, booking)
	}

	span.SetAttributes(attribute.Int("guest.bookings.count", len(bookings)))
	return bookings, err
}

func (s *guestService) GetBookingByDate(ctx context.Context, document string, date time.Time) (domain.Booking, error) {
	ctx, span := guestServiceTracer.Start(ctx, "GuestService.GetBookingByDate")
	defer span.End()
	span.SetAttributes(
		attribute.String("guest.document_id", document),
		attribute.String("booking.date", date.UTC().Format(time.RFC3339)),
	)

	guest, err := s.GetByDocument(ctx, document)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "guest_lookup_failed")
		return domain.Booking{}, fmt.Errorf("failed to find guest %s: %w", document, err)
	}

	allBookings, err := s.bookingRepo.FindByGuestId(ctx, guest.Id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_repo.find_by_guest_id_failed")
		return domain.Booking{}, fmt.Errorf("failed to find allBookings for guest %s: %w", document, err)
	}
	span.SetAttributes(attribute.Int("guest.bookings.count", len(allBookings)))

	startOfDay := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}

	todayStart := startOfDay(date)
	bookingIdx := -1

	for i := range allBookings {
		if startOfDay(allBookings[i].StayPeriod.CheckIn).Equal(todayStart) {
			bookingIdx = i
			break
		}
	}

	if bookingIdx == -1 {
		err := fmt.Errorf("no booking starting today for guest %s", document)
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_not_found_for_day")
		return domain.Booking{}, err
	}

	selectedBooking := allBookings[bookingIdx]

	stayPeriod, err := domain.NewPeriod(selectedBooking.StayPeriod.CheckIn, selectedBooking.StayPeriod.CheckOut)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_period_mapping_failed")
		return domain.Booking{}, fmt.Errorf("map booking stay period for guest %s: %w", document, err)
	}

	booking, err := domain.NewBooking(
		selectedBooking.Id,
		selectedBooking.MainGuest,
		selectedBooking.NumberOfGuests,
		stayPeriod,
		selectedBooking.CottageName,
		selectedBooking.Status,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "booking_mapping_failed")
		return domain.Booking{}, fmt.Errorf("map booking to domain for guest %s: %w", document, err)
	}

	span.SetAttributes(
		attribute.String("booking.id", booking.Id.Hex()),
		attribute.String("booking.cottage_name", booking.CottageName),
	)

	return booking, nil
}
