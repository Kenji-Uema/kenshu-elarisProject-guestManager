package websocket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	appfakes "github.com/Kenji-Uema/guestManager/internal/app/fakes"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	mqfakes "github.com/Kenji-Uema/guestManager/internal/infra/mq/fakes"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat"
	handlerfakes "github.com/Kenji-Uema/guestManager/internal/transport/websocket/handler/fakes"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestWsHandle(t *testing.T) {
	t.Run("returns service unavailable when dependencies are missing", func(t *testing.T) {
		ws := &Ws{}
		recorder := httptest.NewRecorder()
		ctx, engine := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/ws", nil)
		_ = engine

		ws.Handle(ctx)

		if ctx.Writer.Status() != http.StatusServiceUnavailable {
			t.Fatalf("Handle() status = %d, want %d", ctx.Writer.Status(), http.StatusServiceUnavailable)
		}
	})

	t.Run("runs handlers in order and passes booking through", func(t *testing.T) {
		ws := newTestWs(t)
		checkin := &handlerfakes.CheckinHandler{}
		stay := &handlerfakes.StayHandler{}
		checkout := &handlerfakes.CheckoutHandler{}
		booking := testBooking(t, "cottage-7")
		callOrder := make([]string, 0, 3)

		restore := stubHandlerFactories(checkin, stay, checkout)
		defer restore()

		checkin.HandleFn = func(ctx context.Context) (domain.Booking, error) {
			callOrder = append(callOrder, "checkin")
			return booking, nil
		}
		stay.HandleFn = func(ctx context.Context, gotBooking domain.Booking) error {
			callOrder = append(callOrder, "stay")
			if gotBooking.Id != booking.Id {
				t.Fatalf("stay handler booking = %+v, want %+v", gotBooking, booking)
			}
			return nil
		}
		checkout.HandleFn = func(ctx context.Context, gotBooking domain.Booking) error {
			callOrder = append(callOrder, "checkout")
			if gotBooking.Id != booking.Id {
				t.Fatalf("checkout handler booking = %+v, want %+v", gotBooking, booking)
			}
			return nil
		}

		runWebsocketRequest(t, ws)
		waitFor(t, func() bool {
			return checkin.HandleCallCount == 1 && stay.HandleCallCount == 1 && checkout.HandleCallCount == 1
		})

		if checkin.HandleCallCount != 1 || stay.HandleCallCount != 1 || checkout.HandleCallCount != 1 {
			t.Fatalf("Handle() call counts = checkin:%d stay:%d checkout:%d, want 1 each", checkin.HandleCallCount, stay.HandleCallCount, checkout.HandleCallCount)
		}
		if len(callOrder) != 3 || callOrder[0] != "checkin" || callOrder[1] != "stay" || callOrder[2] != "checkout" {
			t.Fatalf("Handle() call order = %+v, want [checkin stay checkout]", callOrder)
		}
	})

	t.Run("stops after checkin error", func(t *testing.T) {
		ws := newTestWs(t)
		checkin := &handlerfakes.CheckinHandler{}
		stay := &handlerfakes.StayHandler{}
		checkout := &handlerfakes.CheckoutHandler{}
		restore := stubHandlerFactories(checkin, stay, checkout)
		defer restore()

		checkin.HandleFn = func(ctx context.Context) (domain.Booking, error) {
			return domain.Booking{}, errors.New("checkin failed")
		}

		runWebsocketRequest(t, ws)
		waitFor(t, func() bool {
			return checkin.HandleCallCount == 1
		})

		if checkin.HandleCallCount != 1 {
			t.Fatalf("Handle() expected checkin to run once, got %d", checkin.HandleCallCount)
		}
		if stay.HandleCallCount != 0 || checkout.HandleCallCount != 0 {
			t.Fatalf("Handle() expected stay/checkout not to run, got stay:%d checkout:%d", stay.HandleCallCount, checkout.HandleCallCount)
		}
	})

	t.Run("stops after stay error", func(t *testing.T) {
		ws := newTestWs(t)
		checkin := &handlerfakes.CheckinHandler{}
		stay := &handlerfakes.StayHandler{}
		checkout := &handlerfakes.CheckoutHandler{}
		booking := testBooking(t, "cottage-8")
		restore := stubHandlerFactories(checkin, stay, checkout)
		defer restore()

		checkin.HandleFn = func(ctx context.Context) (domain.Booking, error) {
			return booking, nil
		}
		stay.HandleFn = func(ctx context.Context, gotBooking domain.Booking) error {
			return errors.New("stay failed")
		}

		runWebsocketRequest(t, ws)
		waitFor(t, func() bool {
			return checkin.HandleCallCount == 1 && stay.HandleCallCount == 1
		})

		if checkin.HandleCallCount != 1 || stay.HandleCallCount != 1 {
			t.Fatalf("Handle() expected checkin/stay to run once, got checkin:%d stay:%d", checkin.HandleCallCount, stay.HandleCallCount)
		}
		if checkout.HandleCallCount != 0 {
			t.Fatalf("Handle() expected checkout not to run, got %d", checkout.HandleCallCount)
		}
	})
}

func newTestWs(t *testing.T) *Ws {
	t.Helper()

	timeEvents, err := app.NewTimeEventService(&mqfakes.FakeMqConsumer{}, &mqfakes.FakeMqConsumer{})
	if err != nil {
		t.Fatalf("NewTimeEventService() error = %v", err)
	}

	return NewWebsocket(
		&appfakes.ReceptionService{},
		&appfakes.CleaningService{},
		nil,
		&appfakes.NotificationService{},
		timeEvents,
	)
}

func stubHandlerFactories(checkin *handlerfakes.CheckinHandler, stay *handlerfakes.StayHandler, checkout *handlerfakes.CheckoutHandler) func() {
	prevCheckin := newCheckinHandler
	prevStay := newStayHandler
	prevCheckout := newCheckoutHandler

	newCheckinHandler = func(receptionService app.ReceptionService,
		clock port.ClockClient, writer chat.Writer, reader chat.Reader) checkinHandler {
		return checkin
	}
	newStayHandler = func(notificationService app.NotificationService, cleaningService app.CleaningService,
		timeEventService app.TimeEventService, writer chat.Writer, reader chat.Reader) stayHandler {
		return stay
	}
	newCheckoutHandler = func(receptionService app.ReceptionService, clock port.ClockClient,
		writer chat.Writer, reader chat.Reader) checkoutHandler {
		return checkout
	}

	return func() {
		newCheckinHandler = prevCheckin
		newStayHandler = prevStay
		newCheckoutHandler = prevCheckout
	}
}

func runWebsocketRequest(t *testing.T, ws *Ws) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/ws", ws.Handle)

	server := httptest.NewServer(engine)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("websocket dial error = %v", err)
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			t.Fatalf("websocket close error = %v", err)
		}
	}(conn)
}

func testBooking(t *testing.T, cottage string) domain.Booking {
	t.Helper()

	stayPeriod, err := domain.NewPeriod(
		time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewPeriod() error = %v", err)
	}

	booking, err := domain.NewBooking([12]byte{1}, [12]byte{2}, 2, stayPeriod, cottage, "confirmed")
	if err != nil {
		t.Fatalf("NewBooking() error = %v", err)
	}

	return booking
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
