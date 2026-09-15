package routes

// need func to protect routes
import (
	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/config"
	controller "github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/controllers"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/middleware"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/services"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func SetupProtectedRoutes(router *gin.Engine, client *mongo.Client, aiConfig config.AIConfig) {
	generator, generatorErr := services.NewLangChainGeneratorWithConfig(aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)
	router.Use(middleware.AuthMiddleWare())
	// if the token is invalid so the code won't continuously execute
	router.GET("/movie/:imdb_id", controller.GetMovie())
	router.POST("/addmovie", controller.AddMovie())
	router.POST("/logout", controller.LogoutUser())
	router.PATCH("/movies/:imdb_id", controller.UpdateMovie())
	router.DELETE("/movies/:imdb_id", controller.DeleteMovie())

	router.GET("/recommendedmovies", controller.GetRecommendationMovies())
	router.POST("/ai/recommend", controller.RecommendMoviesWithAIConfigured(generator, generatorErr, aiConfig.Timeout))
	router.GET("/ai/conversations", controller.ListAIConversations())
	router.GET("/ai/conversations/:conversation_id", controller.GetAIConversation())
	router.PUT("/ai/feedback/:imdb_id", controller.SaveRecommendationFeedback())
	router.PATCH("/updatereview/:imdb_id", controller.AdminReviewUpdate())
}
