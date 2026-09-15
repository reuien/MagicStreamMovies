package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var Client *mongo.Client
var DatabaseName string

func Connect(mongoDBURI, databaseName string) (*mongo.Client, error) {
	if mongoDBURI == "" {
		return nil, fmt.Errorf("MONGODB_URI is not set")
	}
	if databaseName == "" {
		return nil, fmt.Errorf("DATABASE_NAME is not set")
	}
	DatabaseName = databaseName

	clientOptions := options.Client().ApplyURI(mongoDBURI)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect to MongoDB: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	Client = client
	return client, nil
}

func OpenCollection(collectionName string) *mongo.Collection {
	if Client == nil {
		panic("database client is not initialized")
	}
	return Client.Database(DatabaseName).Collection(collectionName)
}

func EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{collection: "users", models: []mongo.IndexModel{{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)}}},
		{collection: "movies", models: []mongo.IndexModel{{Keys: bson.D{{Key: "imdb_id", Value: 1}}, Options: options.Index().SetUnique(true)}}},
		{collection: "recommendation_feedback", models: []mongo.IndexModel{{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "imdb_id", Value: 1}}, Options: options.Index().SetUnique(true)}}},
		{collection: "conversations", models: []mongo.IndexModel{{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "updated_at", Value: -1}}}}},
		{collection: "ai_invocation_audits", models: []mongo.IndexModel{{Keys: bson.D{{Key: "created_at", Value: -1}}}}},
	}
	for _, entry := range indexes {
		if _, err := OpenCollection(entry.collection).Indexes().CreateMany(ctx, entry.models); err != nil {
			return fmt.Errorf("create indexes for %s: %w", entry.collection, err)
		}
	}
	return nil
}
