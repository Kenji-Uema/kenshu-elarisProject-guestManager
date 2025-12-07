package mdb

import (
	"context"
	"encoding/json"
	"guestManager/internal/domain"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoC      *mongodb.MongoDBContainer
	mongoClient *mongo.Client
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var err error
	mongoC, err = mongodb.Run(
		ctx,
		"mongo:latest",
		mongodb.WithUsername("test_user"),
		mongodb.WithPassword("test_pass"),
	)
	if err != nil {
		log.Fatalf("failed to start MongoDB container: %v", err)
	}

	uri, err := mongoC.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("failed to connect mongo client: %v", err)
	}

	code := m.Run()

	_ = mongoClient.Disconnect(ctx)
	_ = testcontainers.TerminateContainer(mongoC)

	os.Exit(code)
}

func setupAndRun(testName string, t *testing.T, test func(t *testing.T, ct *mongo.Collection, br *mongo.Collection)) {
	db := mongoClient.Database("test_db")

	cottageCollection := db.Collection("Cottage")
	bookingCollection := db.Collection("Booking")
	guestCollection := db.Collection("Guest")

	seed[domain.Cottage](t, cottageCollection, "../../test_data/cottages_fixture.json")
	seed[domain.Booking](t, bookingCollection, "../../test_data/bookings_fixture.json")
	seed[domain.Guest](t, guestCollection, "../../test_data/guests_fixture.json")

	t.Cleanup(func() {
		_ = cottageCollection.Drop(context.Background())
		_ = bookingCollection.Drop(context.Background())
		_ = guestCollection.Drop(context.Background())
	})

	t.Run(testName, func(t *testing.T) {
		test(t, cottageCollection, bookingCollection)
	})
}

func seed[D domain.Booking | domain.Cottage | domain.Guest](t *testing.T, collection *mongo.Collection, filepath string) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	var items []D
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatal(err)
	}

	docs := make([]interface{}, len(items))
	for i, g := range items {
		docs[i] = g
	}
	if _, err := collection.InsertMany(context.Background(), docs); err != nil {
		t.Fatal(err)
	}
}
