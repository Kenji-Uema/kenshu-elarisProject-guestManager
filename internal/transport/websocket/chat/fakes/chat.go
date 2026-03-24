package fakes

import (
	"context"
	"errors"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

var ErrNoMoreMessages = errors.New("no more messages")

type Chat struct {
	Messages           []*dto.ChatMessage
	Acked              []*dto.ChatMessage
	Written            []*dto.ChatMessage
	ReadErr            error
	WriteErr           error
	WaitForContextDone bool
	OnReadAck          func(ctx context.Context) (*dto.ChatMessage, error)
	OnReadReply        func(ctx context.Context) (*dto.ChatMessage, error)
	OnWrite            func(msg *dto.ChatMessage)
}

func (f *Chat) ReadOneMessage(ctx context.Context) (*dto.ChatMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.ReadErr != nil {
		return nil, f.ReadErr
	}
	if len(f.Messages) == 0 {
		if f.WaitForContextDone {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return nil, ErrNoMoreMessages
	}

	msg := f.Messages[0]
	f.Messages = f.Messages[1:]
	return msg, nil
}

func (f *Chat) ReadAck(ctx context.Context) (*dto.ChatMessage, error) {
	if f.OnReadAck != nil {
		return f.OnReadAck(ctx)
	}
	return f.ReadOneMessage(ctx)
}

func (f *Chat) ReadReply(ctx context.Context) (*dto.ChatMessage, error) {
	if f.OnReadReply != nil {
		return f.OnReadReply(ctx)
	}
	return f.ReadOneMessage(ctx)
}

func (f *Chat) SendAck(_ context.Context, msg *dto.ChatMessage) {
	if msg == nil || msg.GetAck() != nil {
		return
	}

	f.Acked = append(f.Acked, msg)
}

func (f *Chat) WriteOneMessage(ctx context.Context, msg *dto.ChatMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.WriteErr != nil {
		return f.WriteErr
	}

	f.Written = append(f.Written, msg)
	if f.OnWrite != nil {
		f.OnWrite(msg)
	}

	return nil
}
