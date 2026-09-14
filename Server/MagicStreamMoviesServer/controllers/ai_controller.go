package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/services"
)

func requestContext(c *gin.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), timeout)
}

func RecommendMoviesWithAI() gin.HandlerFunc {
	generator, generatorErr := services.NewLangChainGenerator()

	return func(c *gin.Context) {
		if generatorErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI recommendation service is not configured"})
			return
		}

		var request models.RecommendationRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message must contain 2 to 1000 characters"})
			return
		}

		ctx, cancel := requestContext(c, 30*time.Second)
		defer cancel()
		service := services.NewRecommendationService(movieStore(), generator)
		response, err := service.Recommend(ctx, request.Message)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, ctx.Err()) {
				status = http.StatusGatewayTimeout
			}
			c.JSON(status, gin.H{"error": "unable to generate movie recommendations"})
			return
		}
		c.JSON(http.StatusOK, response)
	}
}
