package app

import (
	"context"
	"guestManager/internal/domain"
	"guestManager/internal/port"
	"reflect"
	"testing"
)

func TestNewCleaningService(t *testing.T) {
	type args struct {
		publisher port.MqPublisher
	}
	tests := []struct {
		name string
		args args
		want CleaningService
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCleaningService(tt.args.publisher); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCleaningService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cleaningService_CleanRoom(t *testing.T) {
	type fields struct {
		publisher port.MqPublisher
	}
	type args struct {
		ctx context.Context
		r   domain.CleaningRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cleaningService{
				publisher: tt.fields.publisher,
			}
			if err := c.CleanRoom(tt.args.ctx, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("CleanRoom() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_cleaningService_MakeupRoom(t *testing.T) {
	type fields struct {
		publisher port.MqPublisher
	}
	type args struct {
		ctx context.Context
		r   domain.CleaningRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cleaningService{
				publisher: tt.fields.publisher,
			}
			if err := c.MakeupRoom(tt.args.ctx, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("MakeupRoom() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
