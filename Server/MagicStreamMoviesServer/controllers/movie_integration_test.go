//go:build integration

package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
)

func TestMoviePaginationUpdateAndDeleteIntegration(t *testing.T) {
	integrationUsers(t)
	movies := movieStore()
	documents := []any{
		models.Movie{ImdbID: "tt-1", Title: "Alpha", Genre: []models.Genre{{GenreName: "Sci-Fi"}}, Ranking: models.Ranking{RankingValue: 8}},
		models.Movie{ImdbID: "tt-2", Title: "Beta", Genre: []models.Genre{{GenreName: "Drama"}}, Ranking: models.Ranking{RankingValue: 7}},
		models.Movie{ImdbID: "tt-3", Title: "Gamma", Genre: []models.Genre{{GenreName: "Sci-Fi"}}, Ranking: models.Ranking{RankingValue: 9}},
	}
	if _, err := movies.InsertMany(context.Background(), documents); err != nil {
		t.Fatalf("insert movies: %v", err)
	}

	listRouter := gin.New()
	listRouter.GET("/movies", GetMovies())
	listRecorder := httptest.NewRecorder()
	listRouter.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/movies?genre=Sci-Fi&page_size=1&sort=title_asc", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var page struct {
		Items      []models.Movie `json:"items"`
		Total      int64          `json:"total"`
		TotalPages int            `json:"total_pages"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode movie page: %v", err)
	}
	if len(page.Items) != 1 || page.Total != 2 || page.TotalPages != 2 || page.Items[0].Title != "Alpha" {
		t.Fatalf("unexpected page: %#v", page)
	}

	adminRouter := gin.New()
	adminRouter.Use(func(c *gin.Context) { c.Set("role", "ADMIN"); c.Next() })
	adminRouter.PATCH("/movies/:imdb_id", UpdateMovie())
	adminRouter.DELETE("/movies/:imdb_id", DeleteMovie())
	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPatch, "/movies/tt-1", strings.NewReader(`{"title":"Updated Alpha"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	adminRouter.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	deleteRecorder := httptest.NewRecorder()
	adminRouter.ServeHTTP(deleteRecorder, httptest.NewRequest(http.MethodDelete, "/movies/tt-1", nil))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", deleteRecorder.Code)
	}
}
