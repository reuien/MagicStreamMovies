package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/routes"
)

func main() {
	router := gin.Default()
	router.GET("/hello", func(c *gin.Context) {
		c.String(200, "hello magic stream movies!")
	})
	// use seperating protected routes  so we can configure the different
	// categaries of our routers

	client, err := database.Connect()
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("database disconnect failed: %v", err)
		}
	}()
	indexContext, cancelIndexes := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelIndexes()
	if err := database.EnsureIndexes(indexContext); err != nil {
		log.Fatalf("database index initialization failed: %v", err)
	}

	routes.SetupUnprotectedRoutes(router)
	routes.SetupProtectedRoutes(router, client)
	if err := router.Run(":8080"); err != nil {
		log.Printf("failed to start server: %v", err)
	}
}
