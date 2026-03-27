package helpers

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func SeedMongoFromFixtures(ctx context.Context, mongoHost string, database string, collectionToFixture map[string]string) error {
	seedCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(fmt.Sprintf("mongodb://test_user:test_pass@%s", mongoHost)))
	if err != nil {
		return fmt.Errorf("connect mongo for fixture seed: %w", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	db := client.Database(database)
	for collectionName, fixturePath := range collectionToFixture {
		if err := seedCollectionFromFixture(seedCtx, db.Collection(collectionName), fixturePath); err != nil {
			return err
		}
	}

	return nil
}

func seedCollectionFromFixture(ctx context.Context, collection *mongo.Collection, fixturePath string) error {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return fmt.Errorf("read fixture %q: %w", fixturePath, err)
	}

	var rawDocs []bson.M
	if err := bson.UnmarshalExtJSON(data, false, &rawDocs); err != nil {
		return fmt.Errorf("unmarshal fixture %q: %w", fixturePath, err)
	}

	docs := make([]any, 0, len(rawDocs))
	for _, doc := range rawDocs {
		docs = append(docs, doc)
	}

	if _, err := collection.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear collection %q before seeding: %w", collection.Name(), err)
	}
	if len(docs) == 0 {
		return nil
	}

	if _, err := collection.InsertMany(ctx, docs); err != nil {
		return fmt.Errorf("insert fixture docs into %q: %w", collection.Name(), err)
	}

	return nil
}
