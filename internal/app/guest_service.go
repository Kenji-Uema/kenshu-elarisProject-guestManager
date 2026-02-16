package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GuestService interface {
	GetById(ctx context.Context, id bson.ObjectID) (domain.Guest, error)
	GetByDocument(ctx context.Context, documentId string) (domain.Guest, error)
	Add(ctx context.Context, guest domain.Guest) (bson.ObjectID, error)
	Update(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error)
	GetBookings(ctx context.Context, guestId bson.ObjectID) ([]domain.Booking, error)
}

type guestService struct {
	guestRepo   port.GuestRepo
	bookingRepo port.BookingRepo
}

func NewGuestService(guestRepo port.GuestRepo, bookingRepo port.BookingRepo) GuestService {
	return &guestService{guestRepo: guestRepo, bookingRepo: bookingRepo}
}

func (s *guestService) GetById(ctx context.Context, id bson.ObjectID) (domain.Guest, error) {
	guestDoc, err := s.guestRepo.GetById(ctx, id)
	if err != nil {
		return domain.Guest{}, err
	}

	guest, err := domain.NewGuest(guestDoc.Id, guestDoc.DocumentId, guestDoc.GivenNames, guestDoc.Surname,
		guestDoc.Email, guestDoc.CreatedAt, guestDoc.LastUpdate)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			return domain.Guest{}, err
		}

		return domain.Guest{}, fmt.Errorf("guestService.GetById(%s): mapping to domain: %w", id.Hex(), err)
	}

	return guest, nil
}

func (s *guestService) GetByDocument(ctx context.Context, documentId string) (domain.Guest, error) {
	guestDoc, err := s.guestRepo.GetByDocument(ctx, documentId)
	if err != nil {
		return domain.Guest{}, err
	}

	guest, err := domain.NewGuest(guestDoc.Id, guestDoc.DocumentId, guestDoc.GivenNames, guestDoc.Surname,
		guestDoc.Email, guestDoc.CreatedAt, guestDoc.LastUpdate)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			return domain.Guest{}, err
		}

		return domain.Guest{}, fmt.Errorf("guestService.GetByDocument(%s): mapping to domain: %w", documentId, err)
	}

	return guest, nil

}

func (s *guestService) Add(ctx context.Context, guest domain.Guest) (bson.ObjectID, error) {
	return s.guestRepo.Add(ctx, guest.ToMongoDoc())
}

func (s *guestService) Update(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error) {
	updatedGuestDoc, err := s.guestRepo.Update(ctx, id, guest.ToMongoDoc())
	if err != nil {
		return domain.Guest{}, err
	}

	updatedGuest, err := domain.NewGuest(updatedGuestDoc.Id, updatedGuestDoc.DocumentId, updatedGuestDoc.GivenNames,
		updatedGuestDoc.Surname, updatedGuestDoc.Email, updatedGuestDoc.CreatedAt, updatedGuestDoc.LastUpdate)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			return domain.Guest{}, err
		}

		return domain.Guest{}, fmt.Errorf("guestService.Update(%s): mapping to domain: %w", id.Hex(), err)
	}

	return updatedGuest, nil
}

func (s *guestService) GetBookings(ctx context.Context, guestId bson.ObjectID) ([]domain.Booking, error) {
	bookingsDoc, err := s.bookingRepo.FindByGuestId(ctx, guestId)
	if err != nil {
		return nil, err
	}

	if len(bookingsDoc) == 0 {
		return []domain.Booking{}, nil
	}

	bookings := make([]domain.Booking, len(bookingsDoc))
	for _, b := range bookingsDoc {
		stayPeriod, err := domain.NewPeriod(b.StayPeriod.Start, b.StayPeriod.End)
		if err != nil {
			return nil, err
		}
		booking, err := domain.NewBooking(
			b.MainGuest,
			b.NumberOfGuests,
			stayPeriod,
			b.CottageName,
			b.Status,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}

	return bookings, err
}
