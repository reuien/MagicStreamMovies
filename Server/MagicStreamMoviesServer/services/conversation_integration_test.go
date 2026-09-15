//go:build integration

package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func integrationConversationService(t *testing.T) (*ConversationService, *mongo.Database) {
	t.Helper()
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping MongoDB: %v", err)
	}
	database := client.Database(fmt.Sprintf("magic_stream_integration_%d", time.Now().UnixNano()))
	feedback := database.Collection("recommendation_feedback")
	if _, err := feedback.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "imdb_id", Value: 1}}, Options: options.Index().SetUnique(true),
	}); err != nil {
		t.Fatalf("create feedback index: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_ = database.Drop(cleanupCtx)
		_ = client.Disconnect(cleanupCtx)
	})
	return NewConversationService(database.Collection("conversations"), feedback), database
}

func TestConversationOwnershipIntegration(t *testing.T) {
	service, _ := integrationConversationService(t)
	ctx := context.Background()
	response := models.RecommendationResponse{Recommendations: []models.MovieRecommendation{{Movie: models.Movie{ImdbID: "tt-1", Title: "Movie"}}}}
	id, err := service.SaveTurn(ctx, "owner", "", "recommend a movie", response)
	if err != nil {
		t.Fatalf("SaveTurn() error = %v", err)
	}
	if _, err := service.Get(ctx, "owner", id); err != nil {
		t.Fatalf("owner Get() error = %v", err)
	}
	if _, err := service.Get(ctx, "other-user", id); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("cross-user Get() error = %v, want ErrConversationNotFound", err)
	}
}

func TestConcurrentFeedbackUpsertIntegration(t *testing.T) {
	service, database := integrationConversationService(t)
	ctx := context.Background()
	const workers = 20
	errCh := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errCh <- service.SaveFeedback(ctx, "user-1", "tt-1", "dislike")
		}()
	}
	wait.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("SaveFeedback() error = %v", err)
		}
	}
	count, err := database.Collection("recommendation_feedback").CountDocuments(ctx, bson.M{"user_id": "user-1", "imdb_id": "tt-1"})
	if err != nil || count != 1 {
		t.Fatalf("feedback count = %d, error = %v", count, err)
	}
	ids, err := service.DislikedMovieIDs(ctx, "user-1")
	if err != nil || len(ids) != 1 || ids[0] != "tt-1" {
		t.Fatalf("DislikedMovieIDs() = %#v, error = %v", ids, err)
	}
}
