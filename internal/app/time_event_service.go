package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type timeEventType string

type TimeEventType = timeEventType

const TimeEventHourChange timeEventType = "hour_change"
const TimeEventDayChange timeEventType = "day_change"

type TimeEventService interface {
	Start(ctx context.Context)
	Register(eventType TimeEventType, ch chan<- time.Time)
	Unregister(eventType TimeEventType, ch chan<- time.Time)
}

type timeEventService struct {
	hourChangeClient port.MqConsumer
	dayChangeClient  port.MqConsumer

	hourChangeChannelsMu sync.RWMutex
	hourChangeChannels   *domain.Set[chan<- time.Time]
	dayChangeChannelsMu  sync.RWMutex
	dayChangeChannels    *domain.Set[chan<- time.Time]
}

func NewTimeEventService(hourChangeClient port.MqConsumer, dayChangeClient port.MqConsumer) (TimeEventService, error) {
	if err := validation.New().
		NotZeroValue("hourChangeClient", hourChangeClient).
		NotZeroValue("dayChangeClient", dayChangeClient).
		Validate(); err != nil {
		return nil, err
	}

	return &timeEventService{
		hourChangeClient:   hourChangeClient,
		dayChangeClient:    dayChangeClient,
		hourChangeChannels: domain.NewSet[chan<- time.Time](),
		dayChangeChannels:  domain.NewSet[chan<- time.Time](),
	}, nil
}

func (s *timeEventService) Start(ctx context.Context) {
	hourChangeDeliveries, err := s.hourChangeClient.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume hour change events", "error", err)
		return
	}

	dayChangeDeliveries, err := s.dayChangeClient.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume day change events", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case hourChange, ok := <-hourChangeDeliveries:
			if !ok {
				slog.InfoContext(ctx, "hour change consumer closed")
				return
			}

			currentTime, err := unmarshalTimeEvent(ctx, hourChange.Body)
			if err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal hour change event", "error", err)
				s.nackDelivery(ctx, hourChange, "hourChangeEvent")
				continue
			}

			s.ackDelivery(ctx, hourChange, "hourChangeEvent")
			s.notifyHourChange(ctx, currentTime)
		case dayChange, ok := <-dayChangeDeliveries:
			if !ok {
				slog.InfoContext(ctx, "day change consumer closed")
				return
			}

			currentTime, err := unmarshalTimeEvent(ctx, dayChange.Body)
			if err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal day change event", "error", err)
				s.nackDelivery(ctx, dayChange, "dayChangeEvent")
				continue
			}

			s.ackDelivery(ctx, dayChange, "dayChangeEvent")
			s.notifyDayChange(ctx, currentTime)
		}
	}
}

func (s *timeEventService) Register(eventType timeEventType, ch chan<- time.Time) {
	if ch == nil {
		return
	}

	switch eventType {
	case TimeEventHourChange:
		s.hourChangeChannelsMu.Lock()
		defer s.hourChangeChannelsMu.Unlock()
		s.hourChangeChannels.Add(ch)
	case TimeEventDayChange:
		s.dayChangeChannelsMu.Lock()
		defer s.dayChangeChannelsMu.Unlock()
		s.dayChangeChannels.Add(ch)
	}
}

func (s *timeEventService) Unregister(eventType timeEventType, ch chan<- time.Time) {
	if ch == nil {
		return
	}

	switch eventType {
	case TimeEventHourChange:
		s.hourChangeChannelsMu.Lock()
		defer s.hourChangeChannelsMu.Unlock()
		s.hourChangeChannels.Remove(ch)
	case TimeEventDayChange:
		s.dayChangeChannelsMu.Lock()
		defer s.dayChangeChannelsMu.Unlock()
		s.dayChangeChannels.Remove(ch)
	}
}

func (s *timeEventService) dayChangeNotification(ctx context.Context) <-chan time.Time {
	ch := make(chan time.Time, 1)
	s.Register(TimeEventDayChange, ch)

	go func() {
		<-ctx.Done()
		s.Unregister(TimeEventDayChange, ch)
		close(ch)
	}()

	return ch
}

func (s *timeEventService) hourChangeNotification(ctx context.Context) <-chan time.Time {
	ch := make(chan time.Time, 1)
	s.Register(TimeEventHourChange, ch)

	go func() {
		<-ctx.Done()
		s.Unregister(TimeEventHourChange, ch)
		close(ch)
	}()

	return ch
}

func (s *timeEventService) notifyHourChange(ctx context.Context, currentTime time.Time) {
	s.hourChangeChannelsMu.RLock()
	channels := s.hourChangeChannels.Values()
	s.hourChangeChannelsMu.RUnlock()

	s.notify(ctx, currentTime, channels, "hour change")
}

func (s *timeEventService) notifyDayChange(ctx context.Context, currentTime time.Time) {
	s.dayChangeChannelsMu.RLock()
	channels := s.dayChangeChannels.Values()
	s.dayChangeChannelsMu.RUnlock()

	s.notify(ctx, currentTime, channels, "day change")
}

func (s *timeEventService) notify(ctx context.Context, currentTime time.Time, channels []chan<- time.Time, eventName string) {
	for _, ch := range channels {
		select {
		case ch <- currentTime:
			slog.DebugContext(ctx, "published "+eventName+" event to subscriber", "currentTime", currentTime)
		default:
			slog.WarnContext(ctx, "skipped "+eventName+" event for busy subscriber", "currentTime", currentTime)
		}
	}
}

func (s *timeEventService) ackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "failed to ack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}

func (s *timeEventService) nackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Nack(false, false); err != nil {
		slog.ErrorContext(ctx, "failed to nack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}

func unmarshalTimeEvent(ctx context.Context, body []byte) (time.Time, error) {
	var timeEvent dto.TimeEvent
	if err := proto.Unmarshal(body, &timeEvent); err != nil {
		slog.WarnContext(ctx, "invalid day.changed payload", "error", err)
		return time.Time{}, err

	}

	if err := validation.New().NotZeroValue("time", timeEvent.GetTime()).Validate(); err != nil {
		return time.Time{}, err
	}

	return timeEvent.GetTime().AsTime(), nil
}
