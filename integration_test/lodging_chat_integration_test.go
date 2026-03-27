package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Lodging chat integration", Ordered, func() {
	t := GinkgoT()

	BeforeEach(func() {
		resetFixtureState(t)

		_, err := suiteCottageCollection.UpdateOne(context.Background(), bson.M{"name": "Lake House"}, bson.M{
			"$set": bson.M{
				"current_guest": bson.NilObjectID,
				"key.holder":    enum.KeyHolderCottage,
				"key.number":    "lake-house-key-1",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		suiteClock.Set(time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC))
	})

	It("handles check-in, stay, and checkout over websocket", func() {
		cleaningMessages := startCleaningConsumer(t)

		conn, _, err := websocket.DefaultDialer.Dial(
			fmt.Sprintf("ws://127.0.0.1:%d/lodging/chat", suiteAppPort),
			http.Header{},
		)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = conn.Close() }()

		performCheckinHandshake(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_ENTER_COTTAGE)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_GO_FOR_A_BATH)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_GO_FOR_DINNER)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_GO_TO_SLEEP)).To(Succeed())
		expectAck(conn)

		publishHourChange(t, time.Date(2024, 6, 2, 6, 0, 0, 0, time.UTC))
		expectNotification(conn, dto.SystemNotification_BREAKFAST_READY)

		publishDayChange(t, time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC))

		Expect(sendGuestAction(conn, dto.GuestAction_WAKEUP)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_GO_FOR_BREAKFAST)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION)).To(Succeed())
		expectAck(conn)

		publishHourChange(t, time.Date(2024, 6, 2, 18, 0, 0, 0, time.UTC))
		expectNotification(conn, dto.SystemNotification_DINNER_READY)

		Expect(sendGuestAction(conn, dto.GuestAction_ENJOY_RESORT)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_GO_FOR_A_BATH)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_GO_FOR_DINNER)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_GO_TO_SLEEP)).To(Succeed())
		expectAck(conn)

		suiteClock.Set(time.Date(2024, 6, 5, 9, 0, 0, 0, time.UTC))
		publishDayChange(t, time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC))
		expectNotification(conn, dto.SystemNotification_CHECK_OUT_TODAY)

		Expect(sendGuestAction(conn, dto.GuestAction_LEAVE_COTTAGE)).To(Succeed())
		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_PROCEED_TO_CHECKOUT)).To(Succeed())
		expectAck(conn)

		requestKey := mustReadChatMessage(conn)
		Expect(requestKey.GetSystemRequest()).To(Equal(dto.SystemRequest_REQUEST_COTTAGE_KEY))
		Expect(sendAck(conn, requestKey)).To(Succeed())
		Expect(sendGuestResponse(conn, requestKey.GetMessageId(), &dto.GuestResponse{
			Payload: &dto.GuestResponse_ReturnCottageKey{
				ReturnCottageKey: &dto.ReturnCottageKey{CottageKeyId: "lake-house-key-1"},
			},
		})).To(Succeed())

		expectAck(conn)

		Expect(sendGuestAction(conn, dto.GuestAction_RETURN_COTTAGE_KEY)).To(Succeed())
		expectAck(conn)

		checkoutComplete := mustReadChatMessage(conn)
		Expect(checkoutComplete.GetSystemNotification()).To(Equal(dto.SystemNotification_CHECK_OUT_COMPLETE))
		Expect(sendAck(conn, checkoutComplete)).To(Succeed())

		Eventually(func(g Gomega) {
			var cottage documents.Cottage
			err := suiteCottageCollection.FindOne(context.Background(), bson.M{"name": "Lake House"}).Decode(&cottage)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cottage.CurrentGuest).To(Equal(bson.NilObjectID))
			g.Expect(cottage.Bookings).To(BeEmpty())
			g.Expect(cottage.Key.Holder).To(Equal(enum.KeyHolderCottage))
			g.Expect(cottage.Key.Number).To(Equal("lake-house-key-1"))

			var booking documents.Booking
			err = suiteBookingCollection.FindOne(context.Background(), bson.M{"_id": mustObjectID(t, "64b6f7c2c0f1e84c0a1a9d01")}).Decode(&booking)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(booking.Status).To(Equal(enum.BookingStatusPast))
		}, "10s", "100ms").Should(Succeed())

		requests := []dto.RequestType{
			mustReadCleaningRequest(cleaningMessages),
			mustReadCleaningRequest(cleaningMessages),
			mustReadCleaningRequest(cleaningMessages),
			mustReadCleaningRequest(cleaningMessages),
			mustReadCleaningRequest(cleaningMessages),
		}
		Expect(requests).To(Equal([]dto.RequestType{
			dto.RequestType_PREPARE_FOR_GUEST,
			dto.RequestType_PREPARE_FOR_SLEEP,
			dto.RequestType_FULL_CLEANING,
			dto.RequestType_PREPARE_FOR_SLEEP,
			dto.RequestType_FULL_CLEANING,
		}))
	})
})

func startCleaningConsumer(t FullGinkgoTInterface) <-chan amqp.Delivery {
	t.Helper()

	channel, err := suiteRabbitConn.Channel()
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Expect(channel.Close()).To(Succeed())
	})

	queue, err := channel.QueueDeclare("", false, true, true, false, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(channel.QueueBind(queue.Name, "", suiteCleaningEx, false, nil)).To(Succeed())

	deliveries, err := channel.Consume(queue.Name, "", false, true, false, false, nil)
	Expect(err).NotTo(HaveOccurred())

	return deliveries
}

func performCheckinHandshake(conn *websocket.Conn) {
	Expect(sendGuestAction(conn, dto.GuestAction_SHOW_FOR_CHECKIN)).To(Succeed())
	expectAck(conn)

	requestDocument := mustReadChatMessage(conn)
	Expect(requestDocument.GetSystemRequest()).To(Equal(dto.SystemRequest_REQUEST_DOCUMENT))
	Expect(sendAck(conn, requestDocument)).To(Succeed())
	Expect(sendGuestResponse(conn, requestDocument.GetMessageId(), &dto.GuestResponse{
		Payload: &dto.GuestResponse_ShowDocument{
			ShowDocument: &dto.ShowDocument{DocumentId: "ID-001"},
		},
	})).To(Succeed())

	expectAck(conn)

	bookingChecking := mustReadChatMessage(conn)
	Expect(bookingChecking.GetSystemNotification()).To(Equal(dto.SystemNotification_BOOKING_CHECKING))
	Expect(sendAck(conn, bookingChecking)).To(Succeed())

	checkinComplete := mustReadChatMessage(conn)
	Expect(checkinComplete.GetSystemNotification()).To(Equal(dto.SystemNotification_CHECK_IN_COMPLETE))
	Expect(sendAck(conn, checkinComplete)).To(Succeed())

	giveKey := mustReadChatMessage(conn)
	Expect(giveKey.GetSystemRequest()).To(Equal(dto.SystemRequest_GIVE_COTTAGE_KEY))
	Expect(sendAck(conn, giveKey)).To(Succeed())
	Expect(sendGuestResponse(conn, giveKey.GetMessageId(), &dto.GuestResponse{
		Payload: &dto.GuestResponse_ReceiveCottageKey{
			ReceiveCottageKey: &dto.ReceiveCottageKey{CottageKeyId: "lake-house-key-1"},
		},
	})).To(Succeed())

	expectAck(conn)

	Expect(sendGuestAction(conn, dto.GuestAction_TAKE_COTTAGE_KEY)).To(Succeed())
	expectAck(conn)
}

func expectAck(conn *websocket.Conn) {
	msg := mustReadChatMessage(conn)
	Expect(msg.GetAck()).NotTo(BeNil())
}

func expectNotification(conn *websocket.Conn, notification dto.SystemNotification) {
	msg := mustReadChatMessage(conn)
	Expect(msg.GetSystemNotification()).To(Equal(notification))
	Expect(sendAck(conn, msg)).To(Succeed())
}

func publishDayChange(t FullGinkgoTInterface, at time.Time) {
	t.Helper()
	publishTimeEvent(t, suiteDayEx, "day.change", at)
}

func publishHourChange(t FullGinkgoTInterface, at time.Time) {
	t.Helper()
	publishTimeEvent(t, suiteHourEx, "hour.change", at)
}

func publishTimeEvent(t FullGinkgoTInterface, exchange string, routingKey string, at time.Time) {
	t.Helper()

	channel, err := suiteRabbitConn.Channel()
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		Expect(channel.Close()).To(Succeed())
	}()

	body, err := proto.Marshal(&dto.TimeEvent{Time: timestamppb.New(at)})
	Expect(err).NotTo(HaveOccurred())

	Expect(channel.PublishWithContext(context.Background(), exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/protobuf",
		Body:        body,
	})).To(Succeed())
}

func mustReadCleaningRequest(deliveries <-chan amqp.Delivery) dto.RequestType {
	select {
	case delivery := <-deliveries:
		var cleaningRequest dto.CleaningRequest
		Expect(proto.Unmarshal(delivery.Body, &cleaningRequest)).To(Succeed())
		Expect(delivery.Ack(false)).To(Succeed())
		return cleaningRequest.GetRequest()
	case <-time.After(5 * time.Second):
		Fail("timed out waiting for cleaning request")
		return dto.RequestType_UNSPECIFIED
	}
}

func mustReadChatMessage(conn *websocket.Conn) *dto.ChatMessage {
	Expect(conn.SetReadDeadline(time.Now().Add(5 * time.Second))).To(Succeed())

	_, body, err := conn.ReadMessage()
	Expect(err).NotTo(HaveOccurred())

	var msg dto.ChatMessage
	Expect(protojson.Unmarshal(body, &msg)).To(Succeed())

	return &msg
}

func sendGuestAction(conn *websocket.Conn, action dto.GuestAction) error {
	return writeChatMessage(conn, &dto.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   uuid.NewString(),
		Sender:          dto.Sender_SENDER_GUEST,
		ProtocolVersion: "lodging.v1",
		Payload: &dto.ChatMessage_GuestAction{
			GuestAction: action,
		},
	})
}

func sendGuestResponse(conn *websocket.Conn, correlationID string, response *dto.GuestResponse) error {
	return writeChatMessage(conn, &dto.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   correlationID,
		Sender:          dto.Sender_SENDER_GUEST,
		ProtocolVersion: "lodging.v1",
		Payload: &dto.ChatMessage_GuestResponse{
			GuestResponse: response,
		},
	})
}

func sendAck(conn *websocket.Conn, acknowledged *dto.ChatMessage) error {
	return writeChatMessage(conn, &dto.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   acknowledged.GetMessageId(),
		Sender:          dto.Sender_SENDER_GUEST,
		ProtocolVersion: acknowledged.GetProtocolVersion(),
		Payload: &dto.ChatMessage_Ack{
			Ack: &dto.Ack{
				AcknowledgedMessageId: acknowledged.GetMessageId(),
				Status:                dto.AckStatus_ACK_STATUS_ACCEPTED,
				Code:                  dto.ErrorCode_ERROR_CODE_NONE,
			},
		},
	})
}

func writeChatMessage(conn *websocket.Conn, msg *dto.ChatMessage) error {
	payload, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}

	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, payload)
}
