package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerAddsRequestIDAndStructuredLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(RequestLogger(slog.New(slog.NewJSONHandler(&output, nil))))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	if recorder.Header().Get(requestIDHeader) == "" {
		t.Fatal("response does not contain a request ID")
	}
	if logLine := output.String(); !strings.Contains(logLine, `"msg":"http_request"`) || !strings.Contains(logLine, `"status":204`) {
		t.Fatalf("unexpected structured log: %s", logLine)
	}
}
