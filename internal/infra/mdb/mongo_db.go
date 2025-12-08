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

type Db struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDb(connectionContext context.Context, uri, dbName string) (*Db, error) {
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

	return &Db{Client: client, Database: client.Database(dbName)}, nil
}

func (d *Db) Close(ctx context.Context) error {
	return d.Client.Disconnect(ctx)
}

func (d *Db) Collection(name string) *mongo.Collection {
	return d.Database.Collection(name)
}
