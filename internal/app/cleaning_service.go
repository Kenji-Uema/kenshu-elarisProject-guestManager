package app

import (
	"context"
	"guestManager/internal/port"
)

type CleaningService interface {
	CleanRoom(ctx context.Context, roomNumber int)
	MakeupRoom(ctx context.Context, roomNumber int)
	PrepareForNewGuest(ctx context.Context, roomNumber int)
}

type cleaningService struct {
	mqClient port.MqClient
}

func NewCleaningService(mqClient port.MqClient) CleaningService {
	return &cleaningService{mqClient: mqClient}
}

func (c cleaningService) CleanRoom(ctx context.Context, roomNumber int) {
	//TODO implement me
	panic("implement me")
}

func (c cleaningService) MakeupRoom(ctx context.Context, roomNumber int) {
	//TODO implement me
	panic("implement me")
}

func (c cleaningService) PrepareForNewGuest(ctx context.Context, roomNumber int) {
	//TODO implement me
	panic("implement me")
}
