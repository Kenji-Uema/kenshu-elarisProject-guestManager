package mdb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

type Mdb struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoDb(connectionContext context.Context, uri, dbName string) (*Mdb, error) {
	connectionContext, connectionCancel := context.WithTimeout(connectionContext, 10*time.Second)
	defer connectionCancel()

	clientOptions := options.Client().ApplyURI(uri).SetMonitor(otelmongo.NewMonitor())
	client, err := mongo.Connect(connectionContext, clientOptions)

	if err != nil {
		return nil, err
	}

	databaseContext, databaseCancel := context.WithTimeout(connectionContext, 5*time.Second)
	defer databaseCancel()

	if err := client.Ping(databaseContext, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping failed for URI: %s, error: %w", uri, err)
	}

	return &Mdb{client: client, database: client.Database(dbName)}, nil
}

func (d *Mdb) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}

func (d *Mdb) Collection(name string) *mongo.Collection {
	return d.database.Collection(name)
}
