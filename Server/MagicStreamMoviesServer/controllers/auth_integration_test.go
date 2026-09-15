//go:build integration

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func integrationUsers(t *testing.T) *mongo.Collection {
	t.Helper()
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
	oldClient, oldName := database.Client, database.DatabaseName
	database.Client = client
	database.DatabaseName = fmt.Sprintf("magic_stream_auth_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_ = client.Database(database.DatabaseName).Drop(cleanupCtx)
		_ = client.Disconnect(cleanupCtx)
		database.Client, database.DatabaseName = oldClient, oldName
	})
	return database.OpenCollection("users")
}

func TestRefreshRotationAndLogoutIntegration(t *testing.T) {
	t.Setenv("SECRET_KEY", "integration-secret")
	gin.SetMode(gin.TestMode)
	users := integrationUsers(t)
	access, refresh, err := utils.GenerateAllTokens("user@example.com", "Test", "User", "USER", "user-1")
	if err != nil {
		t.Fatalf("GenerateAllTokens() error = %v", err)
	}
	if _, err := users.InsertOne(context.Background(), models.User{
		UserID: "user-1", Email: "user@example.com", FirstName: "Test", LastName: "User", Role: "USER", Token: access, RefreshToken: refresh,
	}); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(fmt.Sprintf(`{"refresh_token":%q}`, refresh)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	RefreshTokens()(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var rotated struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if rotated.RefreshToken == refresh || rotated.Token == access {
		t.Fatal("refresh endpoint did not rotate tokens")
	}

	replayRecorder := httptest.NewRecorder()
	replayCtx, _ := gin.CreateTestContext(replayRecorder)
	replayCtx.Request = httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(fmt.Sprintf(`{"refresh_token":%q}`, refresh)))
	replayCtx.Request.Header.Set("Content-Type", "application/json")
	RefreshTokens()(replayCtx)
	if replayRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("replayed refresh status = %d, want %d", replayRecorder.Code, http.StatusUnauthorized)
	}

	logoutRecorder := httptest.NewRecorder()
	logoutRouter := gin.New()
	logoutRouter.Use(func(c *gin.Context) {
		c.Set("userId", "user-1")
		c.Next()
	})
	logoutRouter.POST("/logout", LogoutUser())
	logoutRouter.ServeHTTP(logoutRecorder, httptest.NewRequest(http.MethodPost, "/logout", nil))
	if logoutRecorder.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", logoutRecorder.Code, http.StatusNoContent)
	}
	count, err := users.CountDocuments(context.Background(), bson.M{"user_id": "user-1", "token": bson.M{"$exists": true}})
	if err != nil || count != 0 {
		t.Fatalf("active token count = %d, error = %v", count, err)
	}
}
