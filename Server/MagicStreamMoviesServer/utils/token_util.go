package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"go.mongodb.org/mongo-driver/v2/bson"
)

/*
this is all about login token creating
firstly we need to  get a signed token
we need to get a registered claim
then add token and refresh token field into our user_login model
first we'll generate token and refresh token remember this two kind of thing can not be the same
and a function to handle the update token
*/
type SignedDetails struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	UserId    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func secretKey() ([]byte, error) {
	secret := os.Getenv("SECRET_KEY")
	if secret == "" {
		return nil, errors.New("SECRET_KEY is not set")
	}
	return []byte(secret), nil
}

func GenerateAllTokens(email, firstName, lastname, role, userId string) (string, string, error) {
	key, err := secretKey()
	if err != nil {
		return "", "", err
	}
	now := time.Now().UTC()
	signedToken, err := signToken(key, email, firstName, lastname, role, userId, "access", now, now.Add(15*time.Minute))
	if err != nil {
		return "", "", err
	}
	signedRefreshToken, err := signToken(key, email, firstName, lastname, role, userId, "refresh", now, now.Add(7*24*time.Hour))
	if err != nil {
		return "", "", err
	}
	return signedToken, signedRefreshToken, nil
}

func signToken(key []byte, email, firstName, lastName, role, userID, tokenType string, issuedAt, expiresAt time.Time) (string, error) {
	claims := &SignedDetails{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
		UserId:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "MagicStream",
			Subject:   userID,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
}

func UpdateAllTokens(ctx context.Context, userId, token, refreshToken string) error {
	updateData := bson.M{
		"$set": bson.M{
			"token":         token,
			"refresh_token": refreshToken,
			"updated_at":    time.Now().UTC(),
		},
	}
	_, err := database.OpenCollection("users").UpdateOne(ctx, bson.M{"user_id": userId}, updateData)
	return err
}

func RotateTokens(ctx context.Context, userID, currentRefreshToken, accessToken, refreshToken string) (bool, error) {
	result, err := database.OpenCollection("users").UpdateOne(ctx, bson.M{
		"user_id": userID, "refresh_token": currentRefreshToken,
	}, bson.M{"$set": bson.M{
		"token": accessToken, "refresh_token": refreshToken, "updated_at": time.Now().UTC(),
	}})
	if err != nil {
		return false, err
	}
	return result.ModifiedCount == 1, nil
}

func GetAccessToken(c *gin.Context) (string, error) {
	// reading the token from the header
	authHeader := strings.TrimSpace(c.Request.Header.Get("Authorization"))
	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("Authorization header must use Bearer scheme")
	}
	return parts[1], nil

}

func ValidateToken(tokenString string) (*SignedDetails, error) {
	return validateTokenType(tokenString, "access")
}

func ValidateRefreshToken(tokenString string) (*SignedDetails, error) {
	return validateTokenType(tokenString, "refresh")
}

func validateTokenType(tokenString, expectedType string) (*SignedDetails, error) {
	claims := &SignedDetails{}
	key, err := secretKey()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing algorithm: %s", token.Method.Alg())
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("MagicStream"))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("expected %s token", expectedType)
	}

	return claims, nil
}

func GetRoleFromContext(c *gin.Context) (string, error) {
	role, exists := c.Get("role")
	if !exists {
		return "", errors.New("userRole does not exist in this context !")
	}
	memberRole, okay := role.(string)
	if !okay {
		return "", errors.New("unable to retrieve user_role!")
	}
	return memberRole, nil
}

func GetUserIdFromContext(c *gin.Context) (string, error) {
	userId, exists := c.Get("userId")
	if !exists {
		return "", errors.New("userId does not exist in this context !")
	}
	id, okay := userId.(string)
	if !okay {
		return "", errors.New("unable to retrieve userid!")
	}
	return id, nil
}
