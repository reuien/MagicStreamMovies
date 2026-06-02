package routes

import (
	"github.com/gin-gonic/gin"
	controller "github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/controllers"
)

func SetupUnprotectedRoutes(router *gin.Engine) {

	router.GET("/movies", controller.GetMovies())
	router.POST("/register", controller.RegisterUser())
	router.POST("/login", controller.LoginUser())
}
