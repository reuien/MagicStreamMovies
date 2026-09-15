package apidocs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestEmbeddedDocumentationEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/openapi.yaml", Specification)
	router.GET("/docs", UI)
	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if spec.Code != http.StatusOK || !strings.Contains(spec.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("invalid OpenAPI response: %d %s", spec.Code, spec.Body.String())
	}
	ui := httptest.NewRecorder()
	router.ServeHTTP(ui, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if ui.Code != http.StatusOK || !strings.Contains(ui.Body.String(), "spec-url=\"/openapi.yaml\"") {
		t.Fatalf("invalid docs UI response: %d %s", ui.Code, ui.Body.String())
	}
}
