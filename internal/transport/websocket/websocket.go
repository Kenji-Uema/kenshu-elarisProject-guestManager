package websocket

import (
	"log/slog"
	"net/http"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/handler"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/message"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketServer provides a minimal websocket bootstrap for server-side usage.
type Ws struct {
	upgrader            websocket.Upgrader
	receptionService    app.ReceptionService
	notificationService app.NotificationService
	dayChangeEvent      port.MqConsumer
}

func NewWebsocket(receptionService app.ReceptionService, notificationService app.NotificationService, dayChangeEvent port.MqConsumer) *Ws {
	return &Ws{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Keep bootstrap permissive; tighten origin checks before production.
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		receptionService:    receptionService,
		notificationService: notificationService,
		dayChangeEvent:      dayChangeEvent,
	}
}

// Handle upgrades HTTP requests to websocket and echoes messages back.
func (s *Ws) Handle(c *gin.Context) {
	if s.receptionService == nil || s.notificationService == nil || s.dayChangeEvent == nil {
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

	reader, writer := message.NewExchangeIO(conn)

	checkinHandler := handler.NewCheckinHandler(s.receptionService, writer, reader)
	stayHandler := handler.NewStayHandler(s.notificationService, s.dayChangeEvent, writer, reader)
	checkoutHandler := handler.NewCheckoutHandler(s.receptionService, writer, reader)

	if err := checkinHandler.WaitForGuest(c.Request.Context()); err != nil {
		return
	}

	go func() {
		booking, err := checkinHandler.Handle(c.Request.Context())
		if err != nil {
			return
		}
		err = stayHandler.Handle(c.Request.Context(), booking)
		if err != nil {
			return
		}
		err = checkoutHandler.ProcessCheckout(c.Request.Context())
		if err != nil {
			return
		}
	}()

	reader.Read(c.Request.Context())
}
