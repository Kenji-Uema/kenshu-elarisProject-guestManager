package fakes

import (
	"context"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
)

type TimeEventService struct {
	StartFn      func(ctx context.Context)
	RegisterFn   func(eventType app.TimeEventType, ch chan time.Time)
	UnregisterFn func(eventType app.TimeEventType, ch chan time.Time)

	StartCallCount      int
	RegisterCallCount   int
	UnregisterCallCount int

	LastStartCtx            context.Context
	LastRegisterEventType   app.TimeEventType
	LastRegisterChannel     chan time.Time
	LastUnregisterEventType app.TimeEventType
	LastUnregisterChannel   chan time.Time
}

func (f *TimeEventService) Start(ctx context.Context) {
	f.StartCallCount++
	f.LastStartCtx = ctx

	if f.StartFn != nil {
		f.StartFn(ctx)
	}
}

func (f *TimeEventService) Register(eventType app.TimeEventType, ch chan time.Time) {
	f.RegisterCallCount++
	f.LastRegisterEventType = eventType
	f.LastRegisterChannel = ch

	if f.RegisterFn != nil {
		f.RegisterFn(eventType, ch)
	}
}

func (f *TimeEventService) Unregister(eventType app.TimeEventType, ch chan time.Time) {
	f.UnregisterCallCount++
	f.LastUnregisterEventType = eventType
	f.LastUnregisterChannel = ch

	if f.UnregisterFn != nil {
		f.UnregisterFn(eventType, ch)
	}
}
