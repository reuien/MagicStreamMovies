package middleware

/*
	in this part we need to write the code of checking
	if the user has been appropriately authenticated
*/

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
)

/*
this func is to validate the access or prohibit access
to protect the access
*/
func AuthMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := utils.GetAccessToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			// IF the user not authenticated it won't continue to call the  targeted endpoint
			c.Abort()
			return
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided!"})
			c.Abort()
			return
		}
		claims, err := utils.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token!"})
			c.Abort()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		count, err := database.OpenCollection("users").CountDocuments(ctx, bson.M{"user_id": claims.UserId, "token": token})
		if err != nil || count != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token is expired or revoked"})
			c.Abort()
			return
		}
		c.Set("userId", claims.UserId)
		c.Set("role", claims.Role)

		// continue to execute the targeted endpoint
		c.Next()
	}
}
