package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ port.CottageRepo = (*FakeCottageRepo)(nil)

type FakeCottageRepo struct {
	GetByNameFn          func(ctx context.Context, roomName string) (documents.Cottage, error)
	UpdateCurrentGuestFn func(ctx context.Context, roomName string, guestId bson.ObjectID) error
	UpdateKeyHolderFn    func(ctx context.Context, roomName string, keyNumber string, holder enum.KeyHolder) error
	RemovePastBookingFn  func(ctx context.Context, bookingId bson.ObjectID) error

	GetByNameCallCount          int
	UpdateCurrentGuestCallCount int
	UpdateKeyHolderCallCount    int
	RemovePastBookingCallCount  int

	LastGetByNameCtx           context.Context
	LastRoomName               string
	LastUpdateCurrentGuestCtx  context.Context
	LastUpdateCurrentGuestRoom string
	LastUpdateCurrentGuestId   bson.ObjectID
	LastUpdateKeyHolderCtx     context.Context
	LastUpdateKeyHolderRoom    string
	LastUpdateKeyNumber        string
	LastUpdateKeyHolder        enum.KeyHolder
	LastRemovePastBookingCtx   context.Context
	LastRemovePastBookingId    bson.ObjectID
}

func (f *FakeCottageRepo) GetByName(ctx context.Context, roomName string) (documents.Cottage, error) {
	f.GetByNameCallCount++
	f.LastGetByNameCtx = ctx
	f.LastRoomName = roomName

	if f.GetByNameFn != nil {
		return f.GetByNameFn(ctx, roomName)
	}

	return documents.Cottage{}, nil
}

func (f *FakeCottageRepo) UpdateCurrentGuest(ctx context.Context, roomName string, guestId bson.ObjectID) error {
	f.UpdateCurrentGuestCallCount++
	f.LastUpdateCurrentGuestCtx = ctx
	f.LastUpdateCurrentGuestRoom = roomName
	f.LastUpdateCurrentGuestId = guestId

	if f.UpdateCurrentGuestFn != nil {
		return f.UpdateCurrentGuestFn(ctx, roomName, guestId)
	}

	return nil
}

func (f *FakeCottageRepo) UpdateKeyHolder(ctx context.Context, roomName string, keyNumber string, holder enum.KeyHolder) error {
	f.UpdateKeyHolderCallCount++
	f.LastUpdateKeyHolderCtx = ctx
	f.LastUpdateKeyHolderRoom = roomName
	f.LastUpdateKeyNumber = keyNumber
	f.LastUpdateKeyHolder = holder

	if f.UpdateKeyHolderFn != nil {
		return f.UpdateKeyHolderFn(ctx, roomName, keyNumber, holder)
	}

	return nil
}

func (f *FakeCottageRepo) RemovePastBooking(ctx context.Context, bookingId bson.ObjectID) error {
	f.RemovePastBookingCallCount++
	f.LastRemovePastBookingCtx = ctx
	f.LastRemovePastBookingId = bookingId

	if f.RemovePastBookingFn != nil {
		return f.RemovePastBookingFn(ctx, bookingId)
	}

	return nil
}
