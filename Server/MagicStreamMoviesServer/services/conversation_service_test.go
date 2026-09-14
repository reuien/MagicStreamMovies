package services

import (
	"testing"

	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
)

func TestRecommendationSummary(t *testing.T) {
	response := models.RecommendationResponse{Recommendations: []models.MovieRecommendation{
		{Movie: models.Movie{Title: "星际穿越"}},
		{Movie: models.Movie{Title: "火星救援"}},
	}}
	if got := recommendationSummary(response); got != "推荐：星际穿越、火星救援" {
		t.Fatalf("recommendationSummary() = %q", got)
	}
}

func TestRecommendationSummaryForEmptyResults(t *testing.T) {
	if got := recommendationSummary(models.RecommendationResponse{}); got != "没有找到符合当前条件的电影" {
		t.Fatalf("recommendationSummary() = %q", got)
	}
}
