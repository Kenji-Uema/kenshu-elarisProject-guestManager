package app

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/port"
)

type Dependencies struct {
	GuestRepo             port.GuestRepo
	BookingRepo           port.BookingRepo
	CottageRepo           port.CottageRepo
	CleaningPublisher     port.MqPublisher
	GuestCommunicationPub port.MqPublisher
	HourChangeConsumer    port.MqConsumer
	DayChangeConsumer     port.MqConsumer
	Cache                 port.Cache
}

type Services struct {
	Guest        GuestService
	Cleaning     CleaningService
	Reception    ReceptionService
	TimeEvents   TimeEventService
	Notification NotificationService
	Arrange      ArrangeCottageService
}

func NewServices(ctx context.Context, deps Dependencies) (Services, error) {
	timeEvents, err := NewTimeEventService(deps.HourChangeConsumer, deps.DayChangeConsumer)
	if err != nil {
		return Services{}, err
	}
	go timeEvents.Start(ctx)

	cleaning, err := NewCleaningService(deps.CleaningPublisher)
	if err != nil {
		return Services{}, err
	}

	guest, err := NewGuestService(deps.GuestRepo, deps.BookingRepo)
	if err != nil {
		return Services{}, err
	}

	reception, err := NewReceptionService(
		guest,
		cleaning,
		deps.CottageRepo,
		deps.BookingRepo,
		deps.DayChangeConsumer,
		deps.Cache,
	)
	if err != nil {
		return Services{}, err
	}

	notification, err := NewNotificationService(timeEvents, deps.Cache)
	if err != nil {
		return Services{}, err
	}

	arrange, err := NewArrangeCottageService(deps.BookingRepo, timeEvents, deps.Cache, deps.GuestCommunicationPub)
	if err != nil {
		return Services{}, err
	}
	go arrange.ArrangeCheckIn(ctx)

	return Services{
		Guest:        guest,
		Cleaning:     cleaning,
		Reception:    reception,
		TimeEvents:   timeEvents,
		Notification: notification,
		Arrange:      arrange,
	}, nil
}
