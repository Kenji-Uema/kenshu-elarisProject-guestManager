package websocket

import (
	"log/slog"
	"net/http"

	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/handler"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/message"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Server provides a minimal websocket bootstrap for server-side usage.
type Server struct {
	upgrader websocket.Upgrader
}

func NewServer() *Server {
	return &Server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Keep bootstrap permissive; tighten origin checks before production.
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// RegisterRoutes mounts websocket endpoints on a Gin router/group.
func (s *Server) RegisterRoutes(router gin.IRoutes) {
	router.GET("/ws", s.Handle)
}

// Handle upgrades HTTP requests to websocket and echoes messages back.
func (s *Server) Handle(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "upgrade websocket", "error", err)
		return
	}
	defer conn.Close()

	reader, writer := message.NewExchangeIO(conn)

	checkinHandler := handler.NewCheckinHandler(writer, reader)
	stayHandler := NewStayHandler(writer)
	checkoutHandler := NewCheckoutHandler(writer)

	if err := checkinHandler.WaitForGuest(c.Request.Context()); err != nil {
		return
	}

	go func() {
		if err := checkinHandler.Handle(c.Request.Context()); err != nil {
			return
		}
		stayHandler.Handle()
		checkoutHandler.Handle()
	}()

	reader.Read(c.Request.Context())
}
