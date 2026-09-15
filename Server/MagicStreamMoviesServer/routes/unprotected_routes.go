package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/apidocs"
	controller "github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/controllers"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func SetupUnprotectedRoutes(router *gin.Engine, client *mongo.Client) {

	router.GET("/health", controller.Health())
	router.GET("/ready", controller.Readiness(client))
	router.GET("/openapi.yaml", apidocs.Specification)
	router.GET("/docs", apidocs.UI)
	router.GET("/movies", controller.GetMovies())
	router.POST("/register", controller.RegisterUser())
	router.POST("/login", controller.LoginUser())
	router.POST("/refresh", controller.RefreshTokens())
}
