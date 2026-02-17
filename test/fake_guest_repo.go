package test

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GuestRepoFake struct {
	Data map[bson.ObjectID]documents.Guest
}

func NewGuestRepoFake(data map[bson.ObjectID]documents.Guest) *GuestRepoFake {
	return &GuestRepoFake{Data: data}
}

func (g *GuestRepoFake) GetById(_ context.Context, id bson.ObjectID) (documents.Guest, error) {
	if guest, ok := g.Data[id]; ok {
		return guest, nil
	}

	return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{Id: id}
}

func (g *GuestRepoFake) GetByDocument(_ context.Context, documentId string) (documents.Guest, error) {
	for _, guest := range g.Data {
		if guest.DocumentId == documentId {
			return guest, nil
		}
	}

	return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{DocumentId: documentId}
}

func (g *GuestRepoFake) Add(_ context.Context, newGuest documents.Guest) (bson.ObjectID, error) {
	g.Data[newGuest.Id] = newGuest

	return newGuest.Id, nil
}

func (g *GuestRepoFake) Update(_ context.Context, id bson.ObjectID, updatedGuest documents.Guest) (documents.Guest, error) {
	g.Data[id] = updatedGuest

	return updatedGuest, nil
}
