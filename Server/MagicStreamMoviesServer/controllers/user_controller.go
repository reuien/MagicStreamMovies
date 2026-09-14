package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

func userStore() *mongo.Collection {
	return database.OpenCollection("users")
}

func HashPassword(password string) (string, error) {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashPassword), nil
}

func RegisterUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input data!"})
			return
		}
		validate := validator.New()
		if err := validate.Struct(user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed!", "details": err.Error()})
			return
		}
		hashedPassword, err := HashPassword(user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to hash password !"})
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		count, err := userStore().CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing user!"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		user.UserID = bson.NewObjectID().Hex()
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		user.Password = hashedPassword
		result, err := userStore().InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
		c.JSON(http.StatusCreated, result)
	}
}

func LoginUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userLogin models.UserLogin
		if err := c.ShouldBindJSON(&userLogin); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input data!"})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()
		var foundUser models.User

		err := userStore().FindOne(ctx, bson.M{"email": userLogin.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password !"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(userLogin.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalide email or password !"})
			return
		}

		token, refreshToken, err := utils.GenerateAllTokens(foundUser.Email, foundUser.FirstName, foundUser.LastName, foundUser.Role, foundUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens!"})
			return
		}
		err = utils.UpdateAllTokens(foundUser.UserID, token, refreshToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update token!"})
			return
		}
		c.JSON(http.StatusOK, models.UserResponse{
			UserId:          foundUser.UserID,
			FirstName:       foundUser.FirstName,
			LastName:        foundUser.LastName,
			Email:           foundUser.Email,
			Role:            foundUser.Role,
			Token:           token,
			RefreshToken:    refreshToken,
			FavouriteGenres: foundUser.FavouriteGenres,
		})

	}
}
