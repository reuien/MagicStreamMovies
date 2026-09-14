package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecommendMoviesWithAIReturnsServiceUnavailableWithoutAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/ai/recommend", strings.NewReader(`{"message":"推荐科幻片"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	RecommendMoviesWithAI()(ctx)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
