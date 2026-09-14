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
	Email     string
	FirstName string
	LastName  string
	Role      string
	UserId    string
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
	claims := &SignedDetails{
		Email:     email,
		FirstName: firstName,
		LastName:  lastname,
		Role:      role,
		UserId:    userId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "MagicStream",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * 7 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(key)

	if err != nil {
		return "", "", err // empty for token and empty for freshed token
	}
	// duplicating above

	refreshClaims := &SignedDetails{
		Email:     email,
		FirstName: firstName,
		LastName:  lastname,
		Role:      role,
		UserId:    userId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "MagicStream",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString(key)

	if err != nil {
		return "", "", err // empty for token and empty for freshed token
	}

	return signedToken, signedRefreshToken, nil
}

func UpdateAllTokens(userId, token, refreshToken string) (err error) {
	var ctx, cancel = context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()

	updateAt, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	updateData := bson.M{
		"$set": bson.M{
			"token":        token,
			"refreshToken": refreshToken,
			"updateAt":     updateAt,
		},
	}
	_, err = database.OpenCollection("users").UpdateOne(ctx, bson.M{"user_id": userId}, updateData)

	if err != nil {
		return err
	}
	return nil
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
