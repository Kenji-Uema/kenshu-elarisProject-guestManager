package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kenji-Uema/guestManager/integration_test/helpers"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ = Describe("Guest HTTP integration", Ordered, func() {
	t := GinkgoT()

	BeforeEach(func() {
		resetFixtureState(t)
	})

	It("returns seeded guest", func() {
		guestID := "64b6f7c2c0f1e84c0a1a9c01"
		body := helpers.DoJSON(
			t,
			http.MethodGet,
			fmt.Sprintf("http://127.0.0.1:%d/guest/%s", suiteAppPort, guestID),
			nil,
			http.StatusOK,
		)

		var response map[string]any
		Expect(json.Unmarshal(body, &response)).To(Succeed())
		Expect(response["document_id"]).To(Equal("ID-001"))
		Expect(response["billing_address"]).To(Equal("Mountain Road 11"))
	})

	It("returns guest bookings", func() {
		guestID := "64b6f7c2c0f1e84c0a1a9c01"
		body := helpers.DoJSON(
			t,
			http.MethodGet,
			fmt.Sprintf("http://127.0.0.1:%d/guest/%s/bookings", suiteAppPort, guestID),
			nil,
			http.StatusOK,
		)

		var response []map[string]any
		Expect(json.Unmarshal(body, &response)).To(Succeed())
		Expect(response).To(HaveLen(1))
		Expect(response[0]["cottage_name"]).To(Equal("Lake House"))
		Expect(response[0]["status"]).To(Equal("confirmed"))
	})

	It("creates guest and persists it in mongo", func() {
		suiteClock.Set(time.Date(2024, 6, 2, 9, 30, 0, 0, time.UTC))

		payload := []byte(`{"document_id":"ID-100","given_names":"Taylor","surname":"Stone","email":"taylor.stone@example.com","billing_address":"Ocean Drive 9"}`)
		body := helpers.DoJSON(
			t,
			http.MethodPost,
			fmt.Sprintf("http://127.0.0.1:%d/guest", suiteAppPort),
			payload,
			http.StatusCreated,
		)

		var createdID string
		Expect(json.Unmarshal(body, &createdID)).To(Succeed())
		Expect(createdID).NotTo(BeEmpty())

		var stored documents.Guest
		err := suiteGuestCollection.FindOne(context.Background(), bson.M{"_id": mustObjectID(t, createdID)}).Decode(&stored)
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.DocumentId).To(Equal("ID-100"))
		Expect(stored.BillingAddress).To(Equal("Ocean Drive 9"))
		Expect(stored.CreatedAt.UTC().Format(time.RFC3339)).To(Equal("2024-06-02T09:30:00Z"))
	})

	It("updates guest and preserves created_at from request", func() {
		suiteClock.Set(time.Date(2024, 6, 3, 18, 15, 0, 0, time.UTC))

		guestID := "64b6f7c2c0f1e84c0a1a9c02"
		payload := []byte(`{"document_id":"ID-002","given_names":"Marcus Allen","surname":"Nguyen","email":"marcus.allen@example.com","billing_address":"New City Avenue 99","created_at":"2024-05-30T10:00:00Z"}`)
		body := helpers.DoJSON(
			t,
			http.MethodPatch,
			fmt.Sprintf("http://127.0.0.1:%d/guest/%s", suiteAppPort, guestID),
			payload,
			http.StatusOK,
		)

		var response map[string]any
		Expect(json.Unmarshal(body, &response)).To(Succeed())
		Expect(response["given_names"]).To(Equal("Marcus Allen"))

		var stored documents.Guest
		err := suiteGuestCollection.FindOne(context.Background(), bson.M{"_id": mustObjectID(t, guestID)}).Decode(&stored)
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.GivenNames).To(Equal("Marcus Allen"))
		Expect(stored.Email).To(Equal("marcus.allen@example.com"))
		Expect(stored.CreatedAt.UTC().Format(time.RFC3339)).To(Equal("2024-05-30T10:00:00Z"))
		Expect(stored.LastUpdate).NotTo(BeNil())
		Expect(stored.LastUpdate.UTC().Format(time.RFC3339)).To(Equal("2024-06-03T18:15:00Z"))
	})
})
