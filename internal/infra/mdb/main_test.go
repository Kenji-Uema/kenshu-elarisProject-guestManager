package mdb

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var (
	mongoC            *mongodb.MongoDBContainer
	bookingRepository *bookingRepo
	cottageRepository *cottageRepo
	guestRepository   *guestRepo
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	clientOptions := options.Client().ApplyURI(uri).SetConnectTimeout(10 * time.Second)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatalf("failed to connect mongo client: %v", err)
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		log.Fatalf("failed to connect mongo client: %v", err)
	}

	db := &Mdb{
		client:   client,
		Database: client.Database("test_db"),
	}

	bookingRepository = &bookingRepo{collection: db.Database.Collection("Booking")}
	cottageRepository = &cottageRepo{collection: db.Database.Collection("Cottage")}
	guestRepository = &guestRepo{collection: db.Database.Collection("Guest")}

	code := m.Run()

	_ = db.Close(context.Background())
	_ = testcontainers.TerminateContainer(mongoC)

	os.Exit(code)
}

func setupAndRun(testName string, t *testing.T, test func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection)) {
	cottageCollection := cottageRepository.collection
	bookingCollection := bookingRepository.collection
	guestCollection := guestRepository.collection

	seed[documents.Cottage](t, cottageCollection, "../../../test_data/cottages_fixture.json")
	seed[documents.Booking](t, bookingCollection, "../../../test_data/bookings_fixture.json")
	seed[documents.Guest](t, guestCollection, "../../../test_data/guests_fixture.json")

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
