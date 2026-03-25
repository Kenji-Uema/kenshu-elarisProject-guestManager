package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ port.GuestRepo = (*FakeGuestRepo)(nil)

type FakeGuestRepo struct {
	GetByIdFn       func(ctx context.Context, id bson.ObjectID) (documents.Guest, error)
	GetByDocumentFn func(ctx context.Context, documentId string) (documents.Guest, error)
	AddFn           func(ctx context.Context, newGuest documents.Guest) (bson.ObjectID, error)
	UpdateFn        func(ctx context.Context, id bson.ObjectID, guest documents.Guest) (documents.Guest, error)

	GetByIdCallCount       int
	GetByDocumentCallCount int
	AddCallCount           int
	UpdateCallCount        int

	LastGetByIdCtx       context.Context
	LastGetById          bson.ObjectID
	LastGetByDocumentCtx context.Context
	LastDocumentId       string
	LastAddCtx           context.Context
	LastAddedGuest       documents.Guest
	LastUpdateCtx        context.Context
	LastUpdateId         bson.ObjectID
	LastUpdatedGuest     documents.Guest
}

func (f *FakeGuestRepo) GetById(ctx context.Context, id bson.ObjectID) (documents.Guest, error) {
	f.GetByIdCallCount++
	f.LastGetByIdCtx = ctx
	f.LastGetById = id

	if f.GetByIdFn != nil {
		return f.GetByIdFn(ctx, id)
	}

	return documents.Guest{}, nil
}

func (f *FakeGuestRepo) GetByDocument(ctx context.Context, documentId string) (documents.Guest, error) {
	f.GetByDocumentCallCount++
	f.LastGetByDocumentCtx = ctx
	f.LastDocumentId = documentId

	if f.GetByDocumentFn != nil {
		return f.GetByDocumentFn(ctx, documentId)
	}

	return documents.Guest{}, nil
}

func (f *FakeGuestRepo) Add(ctx context.Context, newGuest documents.Guest) (bson.ObjectID, error) {
	f.AddCallCount++
	f.LastAddCtx = ctx
	f.LastAddedGuest = newGuest

	if f.AddFn != nil {
		return f.AddFn(ctx, newGuest)
	}

	return bson.NilObjectID, nil
}

func (f *FakeGuestRepo) Update(ctx context.Context, id bson.ObjectID, guest documents.Guest) (documents.Guest, error) {
	f.UpdateCallCount++
	f.LastUpdateCtx = ctx
	f.LastUpdateId = id
	f.LastUpdatedGuest = guest

	if f.UpdateFn != nil {
		return f.UpdateFn(ctx, id, guest)
	}

	return documents.Guest{}, nil
}
