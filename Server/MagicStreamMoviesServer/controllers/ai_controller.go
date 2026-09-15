package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/services"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
)

func requestContext(c *gin.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), timeout)
}

func conversationService() *services.ConversationService {
	return services.NewConversationService(database.OpenCollection("conversations"), database.OpenCollection("recommendation_feedback"))
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
		userID, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user identity is unavailable"})
			return
		}
		ctx, cancel := requestContext(c, 30*time.Second)
		defer cancel()
		conversations := conversationService()
		history, err := conversations.Context(ctx, userID, request.ConversationID)
		if err != nil {
			writeConversationError(c, err)
			return
		}
		excludedMovieIDs, err := conversations.DislikedMovieIDs(ctx, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to load recommendation preferences"})
			return
		}
		recommender := services.NewRecommendationService(movieStore(), generator, database.OpenCollection("ai_invocation_audits"))
		response, err := recommender.Recommend(ctx, userID, request.Message, history, excludedMovieIDs)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, ctx.Err()) {
				status = http.StatusGatewayTimeout
			}
			c.JSON(status, gin.H{"error": "unable to generate movie recommendations"})
			return
		}
		conversationID, err := conversations.SaveTurn(ctx, userID, request.ConversationID, request.Message, response)
		if err != nil {
			writeConversationError(c, err)
			return
		}
		response.ConversationID = conversationID
		c.JSON(http.StatusOK, response)
	}
}

func ListAIConversations() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user identity is unavailable"})
			return
		}
		ctx, cancel := requestContext(c, 10*time.Second)
		defer cancel()
		conversations, err := conversationService().List(ctx, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to load conversations"})
			return
		}
		c.JSON(http.StatusOK, conversations)
	}
}

func GetAIConversation() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user identity is unavailable"})
			return
		}
		ctx, cancel := requestContext(c, 10*time.Second)
		defer cancel()
		conversation, err := conversationService().Get(ctx, userID, c.Param("conversation_id"))
		if err != nil {
			writeConversationError(c, err)
			return
		}
		c.JSON(http.StatusOK, conversation)
	}
}

func SaveRecommendationFeedback() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user identity is unavailable"})
			return
		}
		imdbID := c.Param("imdb_id")
		if imdbID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "movie id is required"})
			return
		}
		var request models.RecommendationFeedbackRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type must be like or dislike"})
			return
		}
		ctx, cancel := requestContext(c, 10*time.Second)
		defer cancel()
		if err := conversationService().SaveFeedback(ctx, userID, imdbID, request.Type); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to save recommendation feedback"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "saved"})
	}
}

func writeConversationError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrConversationNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "conversation operation failed"})
}
