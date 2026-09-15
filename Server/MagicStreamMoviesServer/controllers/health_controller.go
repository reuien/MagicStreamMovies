package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func Readiness(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if client == nil || client.Ping(ctx, nil) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "dependency": "mongodb"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
