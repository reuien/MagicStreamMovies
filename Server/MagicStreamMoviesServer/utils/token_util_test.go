package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateToken(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	token, _, err := GenerateAllTokens("user@example.com", "Test", "User", "member", "user-1")
	if err != nil {
		t.Fatalf("GenerateAllTokens() error = %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserId != "user-1" {
		t.Fatalf("ValidateToken() UserId = %q, want %q", claims.UserId, "user-1")
	}
}

func TestValidateTokenRejectsUnexpectedAlgorithm(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, &SignedDetails{
		RegisteredClaims: jwt.RegisteredClaims{Issuer: "MagicStream"},
	})
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	if _, err := ValidateToken(signed); err == nil {
		t.Fatal("ValidateToken() accepted an unexpected signing algorithm")
	}
}

func TestGetAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{name: "valid", header: "Bearer token-value", want: "token-value"},
		{name: "case insensitive", header: "bearer token-value", want: "token-value"},
		{name: "missing scheme", header: "token-value", wantErr: true},
		{name: "empty", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			request.Header.Set("Authorization", tt.header)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = request

			got, err := GetAccessToken(ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetAccessToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("GetAccessToken() = %q, want %q", got, tt.want)
			}
		})
	}
}
