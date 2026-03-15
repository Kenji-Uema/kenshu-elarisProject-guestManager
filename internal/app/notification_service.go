package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/infra/redis"
	"github.com/Kenji-Uema/guestManager/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type NotificationService interface {
	BreakfastNotification(ctx context.Context) <-chan interface{}
	DinnerNotification(ctx context.Context) <-chan interface{}
	CheckOutNotification() <-chan []domain.Booking
}
type notificationService struct {
	hourEventConsumer port.MqConsumer
	redis             *redis.Redis
}

func NewNotificationService(consumer port.MqConsumer, redis *redis.Redis) NotificationService {
	return &notificationService{hourEventConsumer: consumer, redis: redis}
}

func (n notificationService) BreakfastNotification(ctx context.Context) <-chan interface{} {
	return n.consumeHourEvent(ctx, 6)
}

func (n notificationService) DinnerNotification(ctx context.Context) <-chan interface{} {
	return n.consumeHourEvent(ctx, 18)
}

func (n notificationService) CheckOutNotification() <-chan []domain.Booking {
	out := make(chan []domain.Booking)
	ctx := context.Background()

	go func() {
		defer close(out)

		events, err := n.hourEventConsumer.Consume(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "notification service: consume checkout events", "error", err)
			return
		}

		for {
			select {
			case delivery, ok := <-events:
				if !ok {
					return
				}

				eventTime, err := parseTimeEvent(delivery.Body)
				if err != nil {
					n.nackDelivery(ctx, delivery, false)
					continue
				}

				tomorrow := eventTime.AddDate(0, 0, 1).UTC().Format("2006-01-02")
				redisKey := fmt.Sprintf("checkout.%s", tomorrow)

				payload, err := n.redis.Client().Get(ctx, redisKey).Bytes()
				if err != nil {
					if errors.Is(err, goredis.Nil) {
						n.ackDelivery(ctx, delivery)
						continue
					}
					n.nackDelivery(ctx, delivery, true)
					continue
				}

				var checkOutBookings []domain.Booking
				if err := json.Unmarshal(payload, &checkOutBookings); err != nil {
					n.nackDelivery(ctx, delivery, false)
					continue
				}

				select {
				case out <- checkOutBookings:
					n.ackDelivery(ctx, delivery)
				default:
					n.nackDelivery(ctx, delivery, true)
				}
			}
		}
	}()

	return out
}

func (n notificationService) consumeHourEvent(ctx context.Context, targetHour int) <-chan interface{} {
	out := make(chan interface{})

	go func() {
		defer close(out)

		events, err := n.hourEventConsumer.Consume(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "notification service: consume hour events", "error", err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-events:
				if !ok {
					return
				}

				eventTime, err := parseTimeEvent(delivery.Body)
				if err != nil {
					n.nackDelivery(ctx, delivery, false)
					continue
				}

				if eventTime.Hour() != targetHour {
					n.ackDelivery(ctx, delivery)
					continue
				}

				select {
				case out <- eventTime:
					n.ackDelivery(ctx, delivery)
				case <-ctx.Done():
					n.nackDelivery(ctx, delivery, true)
					return
				}
			}
		}
	}()

	return out
}

func parseTimeEvent(body []byte) (time.Time, error) {
	var timeEvent dto.TimeEvent
	if err := proto.Unmarshal(body, &timeEvent); err != nil {
		return time.Time{}, err
	}
	if timeEvent.GetTime() == nil {
		return time.Time{}, errors.New("missing time")
	}
	return timeEvent.GetTime().AsTime(), nil
}

func (n notificationService) ackDelivery(ctx context.Context, delivery amqp.Delivery) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "notification service: ack delivery", "error", err)
	}
}

func (n notificationService) nackDelivery(ctx context.Context, delivery amqp.Delivery, requeue bool) {
	if err := delivery.Nack(false, requeue); err != nil {
		slog.ErrorContext(ctx, "notification service: nack delivery", "error", err, "requeue", requeue)
	}
}
