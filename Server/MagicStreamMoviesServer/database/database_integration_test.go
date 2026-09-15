//go:build integration

package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestEnsureIndexesIntegration(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI is not set")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect MongoDB: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping MongoDB: %v", err)
	}
	oldClient, oldName := Client, DatabaseName
	Client, DatabaseName = client, fmt.Sprintf("magic_stream_indexes_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = client.Database(DatabaseName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		Client, DatabaseName = oldClient, oldName
	})

	if err := EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}
	users := OpenCollection("users")
	if _, err := users.InsertOne(ctx, bson.M{"email": "duplicate@example.com"}); err != nil {
		t.Fatalf("insert first user: %v", err)
	}
	if _, err := users.InsertOne(ctx, bson.M{"email": "duplicate@example.com"}); !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("duplicate email error = %v, want duplicate key", err)
	}
}
