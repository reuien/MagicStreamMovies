package utils

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

var SECRET_KEY string = os.Getenv("SECRET_KEY")
var SECRET_FRESH_KEY string = os.Getenv("SECRET_FRESH_KEY")
var userCollection mongo.Collection = *database.OpenCollection("users")

func GenerateAllTokens(email, firstName, lastname, role, userId string) (string, string, error) {
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
	signedToken, err := token.SignedString([]byte(SECRET_KEY))

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
	signedRefreshToken, err := refreshToken.SignedString([]byte(SECRET_KEY))

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
	_, err = userCollection.UpdateOne(ctx, bson.M{"use_id": userId}, updateData)

	if err != nil {
		return err
	}
	return nil
}

func GetAccessToken(c *gin.Context) (string, error) {
	// reading the token from the header
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header is required !")
	}
	// split the Bearer substr and get a pure jwt token string
	tokenString := authHeader[len("Bearer "):]
	if tokenString == "" {
		return "", errors.New("Bearer token is required !")
	}

	return tokenString, nil

}

func ValidateToken(tokenString string) (*SignedDetails, error) {
	claims := &SignedDetails{}

	// parse a token string into a token object
	// and decode the string into a claims variable
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(SECRET_KEY), nil
	})
	if err != nil {
		return nil, err
	}
	// check if the signature userd HMAC algorithm  ,dismatch return error
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, err
	}
	// if is expired return the error  has expired
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token has expired")
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
