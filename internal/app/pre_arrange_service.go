package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/guestManager/internal/infra/redis"
	"github.com/Kenji-Uema/guestManager/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type PreArrangeService interface {
	PreArrangeCheckin(ctx context.Context)
}

type preArrangeService struct {
	bookingRepo        port.BookingRepo
	clockEventConsumer port.MqConsumer
	redis              *redis.Redis
}

func NewPreArrangeService(bookingRepo port.BookingRepo, clockEventConsumer port.MqConsumer, redisClient *redis.Redis) PreArrangeService {
	return &preArrangeService{
		bookingRepo:        bookingRepo,
		clockEventConsumer: clockEventConsumer,
		redis:              redisClient,
	}
}

func (s preArrangeService) PreArrangeCheckin(ctx context.Context) {
	events, err := s.clockEventConsumer.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume clock events", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-events:
			if !ok {
				slog.WarnContext(ctx, "clock event stream closed")
				return
			}

			var tomorrow time.Time
			if tomorrow, err = s.unmarshalTimeEvent(ctx, delivery.Body); err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal day-change event", "error", err)
				s.nackDelivery(ctx, delivery, false)
				continue
			}

			var bookingDocs []documents.Booking
			if bookingDocs, err = s.bookingRepo.FindByCheckInDate(ctx, tomorrow); err != nil {
				slog.ErrorContext(ctx, "failed to find bookings by check-in date", "date", tomorrow, "error", err)
				s.nackDelivery(ctx, delivery, true)
				continue
			}

			bookings := s.bookingsDocToDomain(ctx, bookingDocs)
			slog.InfoContext(ctx, "parsed bookings for day-change event", "count", len(bookings), "checkInDate", tomorrow)

			if err := s.cacheCheckin(ctx, err, bookingDocs, tomorrow); err != nil {
				s.nackDelivery(ctx, delivery, true)
				continue
			}

			if err := s.cacheCheckout(ctx, bookings); err != nil {
				s.nackDelivery(ctx, delivery, true)
				continue
			}

			slog.InfoContext(ctx, "coming bookings load into redis")
			s.ackDelivery(ctx, delivery)
		}
	}
}

func (s preArrangeService) unmarshalTimeEvent(ctx context.Context, body []byte) (time.Time, error) {
	var timeEvent dto.TimeEvent
	if err := proto.Unmarshal(body, &timeEvent); err != nil {
		slog.WarnContext(ctx, "invalid day.changed payload", "error", err)
		return time.Time{}, err

	}

	if timeEvent.GetTime() == nil {
		slog.WarnContext(ctx, "invalid day.changed payload: missing time")
		return time.Time{}, errors.New("missing time")
	}

	tomorrow := timeEvent.GetTime().AsTime().AddDate(0, 0, 1)
	return tomorrow, nil
}

func (s preArrangeService) bookingsDocToDomain(ctx context.Context, bookingDocs []documents.Booking) []domain.Booking {
	bookings := make([]domain.Booking, 0, len(bookingDocs))
	for _, b := range bookingDocs {
		stayPeriod, err := domain.NewPeriod(b.StayPeriod.Start, b.StayPeriod.End)
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
			} else {
				slog.ErrorContext(ctx, "failed to map booking to domain", "bookingId", b.Id.Hex(), "error", err)
			}
			continue
		}

		bookings = append(bookings, booking)
	}
	return bookings
}

func (s preArrangeService) cacheCheckin(ctx context.Context, err error, bookingDocs []documents.Booking, tomorrow time.Time) error {
	var payload []byte
	if payload, err = json.Marshal(bookingDocs); err != nil {
		slog.ErrorContext(ctx, "failed to marshal check-in bookings", "checkInDate", tomorrow, "error", err)
		return err
	}

	redisKey := fmt.Sprintf("checkin.%s", tomorrow.UTC().Format("2006-01-02"))
	if err := s.redis.Client().Set(ctx, redisKey, payload, 30*time.Minute).Err(); err != nil {
		slog.ErrorContext(ctx, "failed to save check-in bookings in redis", "key", redisKey, "error", err)
		return err
	}
	return nil
}

func (s preArrangeService) cacheCheckout(ctx context.Context, bookings []domain.Booking) error {
	bookingsByCheckOutDate := make(map[string][]domain.Booking)
	for _, booking := range bookings {
		checkOutDate := booking.StayPeriod.End.UTC().Format("2006-01-02")
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
		if err := s.redis.Client().Set(ctx, checkOutKey, payload, 30*time.Minute).Err(); err != nil {
			slog.ErrorContext(ctx, "failed to save check-out bookings in redis", "key", checkOutKey, "error", err)
			cacheCheckoutErr = errors.Join(cacheCheckoutErr, err)
		}
	}

	if cacheCheckoutErr != nil {
		return cacheCheckoutErr
	}
	return nil
}

func (s preArrangeService) ackDelivery(ctx context.Context, delivery amqp.Delivery) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "failed to ack delivery", "error", err, "routingKey", delivery.RoutingKey)
	}
}

func (s preArrangeService) nackDelivery(ctx context.Context, delivery amqp.Delivery, requeue bool) {
	if err := delivery.Nack(false, requeue); err != nil {
		slog.ErrorContext(ctx, "failed to nack delivery", "error", err, "routingKey", delivery.RoutingKey, "requeue", requeue)
	}
}
