package mdb

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

type Mdb struct {
	client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDb(ctx context.Context, config config.MongoConfig) (*Mdb, error) {
	startCtx, cancel := context.WithTimeout(ctx, time.Duration(config.StartupTimeoutInSeconds)*time.Second)
	defer cancel()

	uri := buildMongoURI(config.Username, config.Password, config.Host)

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(otelmongo.NewMonitor(
			otelmongo.WithCommandAttributeDisabled(true),
		)).
		SetConnectTimeout(time.Duration(config.ConnectionTimeoutInSeconds) * time.Second).
		SetServerSelectionTimeout(time.Duration(config.ServerSelectionTimeoutInSeconds) * time.Second).
		SetMaxConnIdleTime(time.Duration(config.MaxConnIdleTimeInSeconds) * time.Second).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize).
		SetRetryWrites(config.RetryWrites)
	if config.ReplicaSet != "" {
		clientOptions.SetReplicaSet(config.ReplicaSet)
	}

	client, err := mongo.Connect(clientOptions)

	if err != nil {
		return nil, err
	}

	databaseContext, databaseCancel := context.WithTimeout(startCtx, time.Duration(config.PingTimeoutInSeconds)*time.Second)
	defer databaseCancel()

	if err := client.Ping(databaseContext, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping failed for URI: %s, error: %w", uri, err)
	}

	return &Mdb{client: client, Database: client.Database(config.Database)}, nil
}

func buildMongoURI(username config.Secret, password config.Secret, host string) string {
	return fmt.Sprintf("mongodb://%s:%s@%s", string(username), string(password), host)
}

func (d *Mdb) Ping() error {
	return d.client.Ping(context.Background(), readpref.Primary())
}

func (d *Mdb) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}
