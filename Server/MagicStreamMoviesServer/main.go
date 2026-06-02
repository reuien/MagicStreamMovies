package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/routes"
	"go.mongodb.org/mongo-driver/v2/mongo"
		"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
)

func main() {
	router := gin.Default()
	router.GET("/hello", func(c *gin.Context) {
		c.String(200, "hello magic stream movies!")
	})
	// use seperating protected routes  so we can configure the different 
	// categaries of our routers 

	var client *mongo.Client = database.Connect()

	routes.SetupUnprotectedRoutes(router)
	routes.SetupProtectedRoutes(router,client)
	if err := router.Run(":8080"); err != nil {
		fmt.Println("Failed to start server", err)
	}
}
