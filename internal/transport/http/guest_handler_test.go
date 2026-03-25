package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appfakes "github.com/Kenji-Uema/guestManager/internal/app/fakes"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	infrafakes "github.com/Kenji-Uema/guestManager/internal/infra/clock/fakes"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGuestHandlerGetGuest(t *testing.T) {
	t.Run("returns guest dto", func(t *testing.T) {
		guest := mustHTTPGuest(t)
		service := &appfakes.GuestService{
			GetByIdFn: func(ctx context.Context, id bson.ObjectID) (domain.Guest, error) {
				return guest, nil
			},
		}
		handler := NewGuestHandler(service, &infrafakes.FakeClockClient{})

		recorder := httptest.NewRecorder()
		ctx, engine := gin.CreateTestContext(recorder)
		engine.GET("/guest/:userId", handler.GetGuest)

		req := httptest.NewRequest(http.MethodGet, "/guest/"+guest.Id.Hex(), nil)
		ctx.Request = req
		ctx.Params = gin.Params{{Key: "userId", Value: guest.Id.Hex()}}

		handler.GetGuest(ctx)

		if recorder.Code != http.StatusOK {
			t.Fatalf("GetGuest() status = %d, want %d", recorder.Code, http.StatusOK)
		}

		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if body["document_id"] != "doc-123" {
			t.Fatalf("document_id = %v, want %q", body["document_id"], "doc-123")
		}
		if body["billing_address"] != "Billing Street 1" {
			t.Fatalf("billing_address = %v, want %q", body["billing_address"], "Billing Street 1")
		}
	})

	t.Run("returns bad request for invalid id", func(t *testing.T) {
		handler := NewGuestHandler(&appfakes.GuestService{}, &infrafakes.FakeClockClient{})

		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/guest/not-an-object-id", nil)
		ctx.Params = gin.Params{{Key: "userId", Value: "not-an-object-id"}}

		handler.GetGuest(ctx)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("GetGuest() status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})
}

func TestGuestHandlerAddGuest(t *testing.T) {
	t.Run("creates guest", func(t *testing.T) {
		service := &appfakes.GuestService{
			AddFn: func(ctx context.Context, guest domain.Guest) (bson.ObjectID, error) {
				return guest.Id, nil
			},
		}
		now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
		clock := &infrafakes.FakeClockClient{
			NowFn: func(ctx context.Context) (*time.Time, error) {
				return &now, nil
			},
		}
		handler := NewGuestHandler(service, clock)

		payload := `{"document_id":"doc-123","given_names":"John","surname":"Doe","email":"john@example.com","billing_address":"Billing Street 1"}`
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/guest", bytes.NewBufferString(payload))
		ctx.Request.Header.Set("Content-Type", "application/json")

		handler.AddGuest(ctx)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("AddGuest() status = %d, want %d", recorder.Code, http.StatusCreated)
		}
		if service.AddCallCount != 1 {
			t.Fatalf("Add() call count = %d, want 1", service.AddCallCount)
		}
		if service.LastAddedGuest.BillingAddress != "Billing Street 1" {
			t.Fatalf("Add() billing address = %q, want %q", service.LastAddedGuest.BillingAddress, "Billing Street 1")
		}
	})

	t.Run("returns internal server error when clock fails", func(t *testing.T) {
		clock := &infrafakes.FakeClockClient{
			NowFn: func(ctx context.Context) (*time.Time, error) {
				return nil, errors.New("clock failed")
			},
		}
		handler := NewGuestHandler(&appfakes.GuestService{}, clock)

		payload := `{"document_id":"doc-123","given_names":"John","surname":"Doe","email":"john@example.com","billing_address":"Billing Street 1"}`
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/guest", bytes.NewBufferString(payload))
		ctx.Request.Header.Set("Content-Type", "application/json")

		handler.AddGuest(ctx)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("AddGuest() status = %d, want %d", recorder.Code, http.StatusInternalServerError)
		}
	})
}

func TestGuestHandlerUpdateGuest(t *testing.T) {
	t.Run("returns updated guest dto", func(t *testing.T) {
		updated := mustHTTPGuest(t)
		service := &appfakes.GuestService{
			UpdateFn: func(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error) {
				return updated, nil
			},
		}
		now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
		clock := &infrafakes.FakeClockClient{
			NowFn: func(ctx context.Context) (*time.Time, error) {
				return &now, nil
			},
		}
		handler := NewGuestHandler(service, clock)

		payload := `{"document_id":"doc-123","given_names":"John","surname":"Doe","email":"john@example.com","billing_address":"Billing Street 1"}`
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPatch, "/guest/"+updated.Id.Hex(), bytes.NewBufferString(payload))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = gin.Params{{Key: "userId", Value: updated.Id.Hex()}}

		handler.UpdateGuest(ctx)

		if recorder.Code != http.StatusOK {
			t.Fatalf("UpdateGuest() status = %d, want %d", recorder.Code, http.StatusOK)
		}

		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if body["billing_address"] != "Billing Street 1" {
			t.Fatalf("billing_address = %v, want %q", body["billing_address"], "Billing Street 1")
		}
	})
}

func TestGuestHandlerGetBookings(t *testing.T) {
	t.Run("returns bookings dto slice", func(t *testing.T) {
		booking := mustHTTPBooking(t)
		service := &appfakes.GuestService{
			GetBookingsFn: func(ctx context.Context, guestId bson.ObjectID) ([]domain.Booking, error) {
				return []domain.Booking{booking}, nil
			},
		}
		handler := NewGuestHandler(service, &infrafakes.FakeClockClient{})

		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/guest/"+booking.MainGuest.Hex()+"/bookings", nil)
		ctx.Params = gin.Params{{Key: "userId", Value: booking.MainGuest.Hex()}}

		handler.GetBookings(ctx)

		if recorder.Code != http.StatusOK {
			t.Fatalf("GetBookings() status = %d, want %d", recorder.Code, http.StatusOK)
		}

		var body []map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(body) != 1 {
			t.Fatalf("GetBookings() returned %d items, want 1", len(body))
		}
		if body[0]["cottage_name"] != booking.CottageName {
			t.Fatalf("cottage_name = %v, want %q", body[0]["cottage_name"], booking.CottageName)
		}
	})
}

func mustHTTPGuest(t *testing.T) domain.Guest {
	t.Helper()
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	guest, err := domain.NewGuest(
		bson.NewObjectID(),
		"doc-123",
		"John",
		"Doe",
		"john@example.com",
		"Billing Street 1",
		&now,
		&now,
	)
	if err != nil {
		t.Fatalf("NewGuest() error = %v", err)
	}
	return guest
}

func mustHTTPBooking(t *testing.T) domain.Booking {
	t.Helper()
	stay, err := domain.NewPeriod(
		time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewPeriod() error = %v", err)
	}
	booking, err := domain.NewBooking(
		bson.NewObjectID(),
		bson.NewObjectID(),
		2,
		stay,
		"cottage-7",
		enum.BookingStatusConfirmed,
	)
	if err != nil {
		t.Fatalf("NewBooking() error = %v", err)
	}
	return booking
}
