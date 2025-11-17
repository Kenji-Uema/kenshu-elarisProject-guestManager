package app

import (
	"context"
	"fmt"
	"guestManager/internal/port"
	"time"
)

type ReceptionService interface {
	CheckIn(ctx context.Context, guestDocument string, today time.Time) error
	CheckOut(ctx context.Context, guestDocument string, today time.Time) error
}

type receptionService struct {
	guestService    GuestService
	cleaningService CleaningService

	cottageRepo port.CottageRepo
	bookingRepo port.BookingRepo
}

func NewReceptionService(guestService GuestService, cleaningService CleaningService,
	cottageRepo port.CottageRepo, bookingRepo port.BookingRepo) ReceptionService {
	return &receptionService{
		guestService:    guestService,
		cleaningService: cleaningService,
		cottageRepo:     cottageRepo,
		bookingRepo:     bookingRepo,
	}
}

func (r receptionService) CheckIn(ctx context.Context, document string, today time.Time) error {
	guest, err := r.guestService.GetByDocument(ctx, document)

	if err != nil {
		return fmt.Errorf("failed to find guest %s: %w", document, err)
	}

	allBookings, err := r.bookingRepo.FindByGuestId(ctx, guest.Id)
	if err != nil {
		return fmt.Errorf("failed to find allBookings for guest %s: %w", document, err)
	}

	startOfDay := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}

	todayStart := startOfDay(today)
	bookingIdx := -1

	for i := range allBookings {
		if startOfDay(allBookings[i].StayPeriod.Start).Equal(todayStart) {
			bookingIdx = i
			break
		}
	}

	if bookingIdx == -1 {
		return fmt.Errorf("no booking starting today for guest %s", document)
	}

	bookedRoomName := allBookings[bookingIdx].CottageName

	if err := r.cottageRepo.UpdateCurrentGuest(ctx, bookedRoomName, guest.Id); err != nil {
		return fmt.Errorf("failed to update current guest for room %s: %w", bookedRoomName, err)
	}

	return nil
}

func (r receptionService) CheckOut(ctx context.Context, guestId string, today time.Time) error {
	//TODO implement me
	panic("implement me")
}
