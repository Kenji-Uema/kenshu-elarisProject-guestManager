package http

import (
	"net/http"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	infrolog "github.com/Kenji-Uema/guestManager/internal/infra/logging"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/http/bindings"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GuestHandler interface {
	GetGuest(c *gin.Context)
	AddGuest(c *gin.Context)
	UpdateGuest(c *gin.Context)
	GetBookings(c *gin.Context)
}

type guestHandler struct {
	service     app.GuestService
	clockClient port.ClockClient
}

func NewGuestHandler(service app.GuestService, clockClient port.ClockClient) GuestHandler {
	return &guestHandler{service: service, clockClient: clockClient}
}

func guestRequestAttrs(c *gin.Context, attrs ...any) []any {
	base := []any{
		"method", c.Request.Method,
		"path", c.FullPath(),
	}

	return append(base, attrs...)
}

func (h guestHandler) GetGuest(c *gin.Context) {
	var guestIdUri bindings.GuestIdURI

	if err := c.ShouldBindUri(&guestIdUri); err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_retrieval_invalid_request", guestRequestAttrs(c, "error", err)...)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_retrieval_started", guestRequestAttrs(c, "guest_id", guestIdUri.Id)...)

	guestId, err := bson.ObjectIDFromHex(guestIdUri.Id)

	if err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_retrieval_invalid_guest_id", guestRequestAttrs(c, "guest_id", guestIdUri.Id, "error", err)...)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_retrieval_guest_id_parsed", guestRequestAttrs(c, "guest_id", guestId.Hex())...)

	guest, err := h.service.GetById(c.Request.Context(), guestId)
	if err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_retrieval_failed", guestRequestAttrs(c, "guest_id", guestIdUri.Id, "error", err)...)
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	guestDTO := guest.ToDto()

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_retrieval_guest_loaded",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"document_id", guestDTO.DocumentId,
			"email", guestDTO.Email,
		)...,
	)

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_retrieval_succeeded",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"document_id", guestDTO.DocumentId,
		)...,
	)

	c.JSON(200, guestDTO)
}

func (h guestHandler) AddGuest(c *gin.Context) {
	var guestRequest dto.GuestDto

	if err := c.ShouldBindJSON(&guestRequest); err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_registration_invalid_request", guestRequestAttrs(c, "error", err)...)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_registration_started",
		guestRequestAttrs(c,
			"document_id", guestRequest.DocumentId,
			"email", guestRequest.Email,
			"given_names", guestRequest.GivenNames,
			"surname", guestRequest.Surname,
		)...,
	)

	createdTime, err := h.clockClient.Now(c.Request.Context())
	if err != nil {
		infrolog.GuestHTTPError(c.Request.Context(), "guest_registration_clock_failed",
			guestRequestAttrs(c,
				"document_id", guestRequest.DocumentId,
				"error", err,
			)...,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get clock time"})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_registration_clock_resolved",
		guestRequestAttrs(c,
			"document_id", guestRequest.DocumentId,
			"created_at", createdTime,
		)...,
	)

	guest, err := domain.NewGuest(
		bson.NewObjectID(),
		guestRequest.DocumentId,
		guestRequest.GivenNames,
		guestRequest.Surname,
		guestRequest.Email,
		guestRequest.BillingAddress,
		createdTime,
		nil,
	)
	if err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_registration_validation_failed",
			guestRequestAttrs(c,
				"document_id", guestRequest.DocumentId,
				"error", err,
			)...,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	guestDTO := guest.ToDto()

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_registration_guest_validated",
		guestRequestAttrs(c,
			"document_id", guestDTO.DocumentId,
			"email", guestDTO.Email,
		)...,
	)

	id, err := h.service.Add(c.Request.Context(), guest)
	if err != nil {
		infrolog.GuestHTTPError(c.Request.Context(), "guest_registration_failed",
			guestRequestAttrs(c,
				"document_id", guestRequest.DocumentId,
				"error", err,
			)...,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_registration_succeeded",
		guestRequestAttrs(c,
			"guest_id", id.Hex(),
			"document_id", guestRequest.DocumentId,
		)...,
	)

	c.JSON(http.StatusCreated, id)
}

func (h guestHandler) UpdateGuest(c *gin.Context) {
	var updatedRequest dto.GuestDto
	var guestIdUri bindings.GuestIdURI

	if err := c.ShouldBindUri(&guestIdUri); err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_update_invalid_request", guestRequestAttrs(c, "error", err)...)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&updatedRequest); err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_update_invalid_request", guestRequestAttrs(c, "guest_id", guestIdUri.Id, "error", err)...)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_update_started",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"document_id", updatedRequest.DocumentId,
			"email", updatedRequest.Email,
		)...,
	)

	targetGuestId, err := bson.ObjectIDFromHex(guestIdUri.Id)

	if err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_update_invalid_guest_id", guestRequestAttrs(c, "guest_id", guestIdUri.Id, "error", err)...)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_update_guest_id_parsed", guestRequestAttrs(c, "guest_id", targetGuestId.Hex())...)

	createdTime, err := h.clockClient.Now(c.Request.Context())
	if err != nil {
		infrolog.GuestHTTPError(c.Request.Context(), "guest_update_clock_failed", guestRequestAttrs(c, "guest_id", guestIdUri.Id, "error", err)...)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get clock time"})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_update_clock_resolved",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"updated_at", createdTime,
		)...,
	)

	initialCreatedAt := updatedRequest.CreatedAt
	if initialCreatedAt == nil {
		initialCreatedAt = createdTime
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_update_timestamps_resolved",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"created_at", initialCreatedAt,
			"last_update", createdTime,
		)...,
	)

	updatedGuest, err := domain.NewGuest(
		targetGuestId,
		updatedRequest.DocumentId,
		updatedRequest.GivenNames,
		updatedRequest.Surname,
		updatedRequest.Email,
		updatedRequest.BillingAddress,
		initialCreatedAt,
		createdTime,
	)
	if err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_update_validation_failed",
			guestRequestAttrs(c,
				"guest_id", guestIdUri.Id,
				"document_id", updatedRequest.DocumentId,
				"error", err,
			)...,
		)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_update_guest_validated",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"document_id", updatedRequest.DocumentId,
		)...,
	)

	resultedGuest, err := h.service.Update(c.Request.Context(), targetGuestId, updatedGuest)

	if err != nil {
		infrolog.GuestHTTPError(c.Request.Context(), "guest_update_failed",
			guestRequestAttrs(c,
				"guest_id", guestIdUri.Id,
				"document_id", updatedRequest.DocumentId,
				"error", err,
			)...,
		)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	resultedGuestDTO := resultedGuest.ToDto()

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_update_guest_persisted",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"document_id", resultedGuestDTO.DocumentId,
			"email", resultedGuestDTO.Email,
		)...,
	)

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_update_succeeded",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"document_id", resultedGuestDTO.DocumentId,
		)...,
	)

	c.JSON(200, resultedGuestDTO)
}

func (h guestHandler) GetBookings(c *gin.Context) {
	var guestIdUri bindings.GuestIdURI

	if err := c.ShouldBindUri(&guestIdUri); err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_bookings_invalid_request", guestRequestAttrs(c, "error", err)...)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_bookings_retrieval_started", guestRequestAttrs(c, "guest_id", guestIdUri.Id)...)

	guestId, err := bson.ObjectIDFromHex(guestIdUri.Id)

	if err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_bookings_invalid_guest_id", guestRequestAttrs(c, "guest_id", guestIdUri.Id, "error", err)...)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_bookings_guest_id_parsed", guestRequestAttrs(c, "guest_id", guestId.Hex())...)

	bookingsDomain, err := h.service.GetBookings(c.Request.Context(), guestId)
	if err != nil {
		infrolog.GuestHTTPWarn(c.Request.Context(), "guest_bookings_retrieval_failed", guestRequestAttrs(c, "guest_id", guestIdUri.Id, "error", err)...)
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_bookings_loaded",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"count", len(bookingsDomain),
		)...,
	)

	bookingsDto := make([]dto.BookingDto, 0, len(bookingsDomain))
	for _, booking := range bookingsDomain {
		bookingsDto = append(bookingsDto, booking.ToDto())
	}

	if len(bookingsDto) > 0 {
		infrolog.GuestHTTPInfo(c.Request.Context(), "guest_bookings_mapped",
			guestRequestAttrs(c,
				"guest_id", guestIdUri.Id,
				"count", len(bookingsDto),
				"first_cottage_name", bookingsDto[0].CottageName,
				"first_status", bookingsDto[0].Status,
			)...,
		)
	}

	infrolog.GuestHTTPInfo(c.Request.Context(), "guest_bookings_retrieval_succeeded",
		guestRequestAttrs(c,
			"guest_id", guestIdUri.Id,
			"count", len(bookingsDto),
		)...,
	)

	c.JSON(200, bookingsDto)
}
