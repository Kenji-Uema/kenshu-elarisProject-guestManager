package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/handler"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketServer provides a minimal websocket bootstrap for server-side usage.
const (
	defaultReplyTimeout = 30 * time.Second
	defaultAckTimeout   = 5 * time.Second
)

type checkinHandler interface {
	Handle(ctx context.Context) (domain.Booking, error)
}

type stayHandler interface {
	Handle(ctx context.Context, booking domain.Booking) error
}

type checkoutHandler interface {
	Handle(ctx context.Context, booking domain.Booking) error
}

var newCheckinHandler = func(receptionService app.ReceptionService, cleaningService app.CleaningService,
	clock port.ClockClient, writer chat.Writer, reader chat.Reader) checkinHandler {
	return handler.NewCheckinHandler(receptionService, cleaningService, clock, writer, reader)
}

var newStayHandler = func(notificationService app.NotificationService, cleaningService app.CleaningService,
	timeEventService app.TimeEventService, writer chat.Writer, reader chat.Reader) stayHandler {
	return handler.NewStayHandler(notificationService, cleaningService, timeEventService, writer, reader)
}

var newCheckoutHandler = func(receptionService app.ReceptionService, clock port.ClockClient,
	writer chat.Writer, reader chat.Reader) checkoutHandler {
	return handler.NewCheckoutHandler(receptionService, clock, writer, reader)
}

type Ws struct {
	upgrader            websocket.Upgrader
	receptionService    app.ReceptionService
	cleaningService     app.CleaningService
	clockClient         port.ClockClient
	notificationService app.NotificationService
	timeEventService    app.TimeEventService
}

func NewWebsocket(receptionService app.ReceptionService, cleaningService app.CleaningService, clockClient port.ClockClient,
	notificationService app.NotificationService, timeEventService app.TimeEventService) *Ws {
	return &Ws{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Keep bootstrap permissive; tighten origin checks before production.
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		receptionService:    receptionService,
		cleaningService:     cleaningService,
		clockClient:         clockClient,
		notificationService: notificationService,
		timeEventService:    timeEventService,
	}
}

// Handle upgrades HTTP requests to websocket and echoes messages back.
func (s *Ws) Handle(c *gin.Context) {
	if s.receptionService == nil || s.notificationService == nil || s.timeEventService == nil {
		slog.ErrorContext(c.Request.Context(), "websocket dependencies not configured")
		c.Status(http.StatusServiceUnavailable)
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "upgrade websocket", "error", err)
		return
	}
	defer conn.Close()

	reader, writer := chat.NewChat(conn, defaultReplyTimeout, defaultAckTimeout)

	checkinHandler := newCheckinHandler(s.receptionService, s.cleaningService, s.clockClient, writer, reader)
	stayHandler := newStayHandler(s.notificationService, s.cleaningService, s.timeEventService, writer, reader)
	checkoutHandler := newCheckoutHandler(s.receptionService, s.clockClient, writer, reader)

	booking, err := checkinHandler.Handle(c.Request.Context())
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "checkin handler", "error", err)
		return
	}

	err = stayHandler.Handle(c.Request.Context(), booking)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "stay handler", "error", err)
		return
	}

	err = checkoutHandler.Handle(c.Request.Context(), booking)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "checkout handler", "error", err)
		return
	}
}
