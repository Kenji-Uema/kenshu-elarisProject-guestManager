package mdb

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	mongoC            *mongodb.MongoDBContainer
	bookingRepository *bookingRepo
	cottageRepository *cottageRepo
	guestRepository   *guestRepo
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

	parsedURI, err := url.Parse(uri)
	if err != nil {
		log.Fatalf("failed to parse connection string: %v", err)
	}
	mongoHost := parsedURI.Host
	if parsedURI.RawQuery != "" {
		mongoHost = fmt.Sprintf("%s/?%s", parsedURI.Host, parsedURI.RawQuery)
	}

	db, err := NewMongoDb(context.Background(), config.MongoConfig{
		Username: "test_user",
		Password: "test_pass",
		Host:     mongoHost,
		Database: "test_db",
	})
	if err != nil {
		log.Fatalf("failed to connect mongo client: %v", err)
	}

	bookingRepository = &bookingRepo{collection: db.Database.Collection("Booking")}
	cottageRepository = &cottageRepo{collection: db.Database.Collection("Cottage")}
	guestRepository = &guestRepo{collection: db.Database.Collection("Guest")}

	code := m.Run()

	if db != nil {
		_ = db.Close(context.Background())
	}
	_ = testcontainers.TerminateContainer(mongoC)

	os.Exit(code)
}

func setupAndRun(testName string, t *testing.T, test func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection)) {
	cottageCollection := cottageRepository.collection
	bookingCollection := bookingRepository.collection
	guestCollection := guestRepository.collection

	seed[documents.Cottage](t, cottageCollection, "../../../test/test_data/cottages_fixture.json")
	seed[documents.Booking](t, bookingCollection, "../../../test/test_data/bookings_fixture.json")
	seed[documents.Guest](t, guestCollection, "../../../test/test_data/guests_fixture.json")

	t.Cleanup(func() {
		_ = cottageCollection.Drop(context.Background())
		_ = bookingCollection.Drop(context.Background())
		_ = guestCollection.Drop(context.Background())
	})

	t.Run(testName, func(t *testing.T) {
		test(t, cottageCollection, bookingCollection, guestCollection)
	})
}

func seed[D documents.Booking | documents.Cottage | documents.Guest](t *testing.T, collection *mongo.Collection, filepath string) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	var items []D
	if err := bson.UnmarshalExtJSON(data, false, &items); err != nil {
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
