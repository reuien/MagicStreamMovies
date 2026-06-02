package controllers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
	"github.com/tmc/langchaingo/llms/openai"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var movieCollection mongo.Collection = *database.OpenCollection("movies")
var rankingCollection mongo.Collection = *database.OpenCollection("rankings")
var validate = validator.New()

func GetMovies() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var movies []models.Movie

		curcor, err := movieCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies like that !"})
		}
		defer curcor.Close(ctx)

		if err := curcor.All(ctx, &movies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error ": "failed to decoded movies"})
		}
		c.JSON(http.StatusOK, movies)
	}
}

func GetMovie() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		movieID := c.Param("imdb_id")
		if movieID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "movie id is required"})
			return
		}
		var movie models.Movie
		// pass a value of context and a filter
		err := movieCollection.FindOne(ctx, bson.M{"imdb_id": movieID}).Decode(&movie)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}
		c.JSON(http.StatusOK, movie)

	}
}

func AddMovie() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var movie models.Movie
		if err := c.ShouldBindJSON(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input! "})
			return
		}
		if err := validate.Struct(movie); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Validation failed", "detailed": err.Error()})
			return
		}
		result, err := movieCollection.InsertOne(ctx, movie)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add movie !"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

// update review for particular movie
// give ai a prompt which would give a feedback comment in return
func AdminReviewUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {

		role,err := utils.GetRoleFromContext(c) 
		if err != nil {
			c.JSON(http.StatusBadRequest,gin.H{"error":"Role not found in the context !"})
			return 
		}
		if role != "ADMIN"{
			c.JSON(http.StatusUnauthorized,gin.H{"error":"User must be part of the admin"})
			return 
		}

		movieId := c.Param("imdb_id") // passed in url a unique identifier
		if movieId == "" {
			//the id is not included in the url
			c.JSON(http.StatusBadRequest, gin.H{"error": "Movie id is required !"})
			return
		}
		var req struct {
			AdminReview string `json:"admin_review"`
		}
		var resp struct {
			RankingName string `json:"ranking_name"`
			AdminReview string `json:"admin_review"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body ! "})
			return
		}

		// use langchain go to connect the openAI or the other AI powered
		// and return the satiment we need with our prompt
		/*
			function has been done. call the GetReviewRanking to get the ai review
		*/
		sentiment, rankVal, err := GetReviewRanking(req.AdminReview)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting review ranking !"})
			return
		}
		// filter the parameter
		filter := bson.M{"imdb_id": movieId}
		update := bson.M{
			"$set": bson.M{
				"admin_review": req.AdminReview,
				"ranking": bson.M{
					"ranking_value": rankVal,
					"ranking_name":  sentiment,
				},
			},
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		result, err := movieCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating movie !"})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found !"})
			return
		}
		resp.RankingName = sentiment
		resp.AdminReview = req.AdminReview
		c.JSON(http.StatusOK, resp)
	}
}

// encapsulating the fuctionality of ai
func GetReviewRanking(admin_review string) (string, int, error) {
	rankings, err := GetRankings()

	if err != nil {
		return "", 0, err
	}
	sentimentDelimited := ""
	for _, ranking := range rankings {
		if ranking.RankingValue != 999 {
			sentimentDelimited = sentimentDelimited + ranking.RankingName + ","
		}
	}
	sentimentDelimited = strings.Trim(sentimentDelimited, ",")
	err = godotenv.Load(".env")
	if err != nil {
		log.Println("Warning ! env file not found !")
	}

	OpenAIApiKey := os.Getenv("OPENAI_API_KEY")
	OpenAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	OpenAIModel := os.Getenv("OPENAI_MODEL")

	if OpenAIApiKey == "" {
		return "", 0, errors.New("could not read OPENAI_API_KEY!")
	}

	opts := []openai.Option{openai.WithToken(OpenAIApiKey)}
	if OpenAIBaseURL != "" {
		opts = append(opts, openai.WithBaseURL(OpenAIBaseURL))
	}
	if OpenAIModel != "" {
		opts = append(opts, openai.WithModel(OpenAIModel))
	}
	llm, err := openai.New(opts...)

	if err != nil {
		return "", 0, err
	}
	base_prompt_template := os.Getenv("BASE_PROMPT_TEMPLATE")
	base_prompt := strings.Replace(base_prompt_template, "{rankings}", sentimentDelimited, 1)
	response, err := llm.Call(context.Background(), base_prompt+admin_review)
	if err != nil {
		return "", 0, err
	}
	rankVal := 0
	for _, ranking := range rankings {
		if ranking.RankingName == response {
			rankVal = ranking.RankingValue
			break
		}
	}
	return response, rankVal, nil
}

func GetRankings() ([]models.Ranking, error) {
	var rankings []models.Ranking

	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	curcor, err := rankingCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer curcor.Close(ctx)

	if err := curcor.All(ctx, &rankings); err != nil {
		return nil, err
	}
	return rankings, nil
}

func GetRecommendationMovies() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User Id NOT FOUND IN CONTEXT !"})
			return
		}

		favourite_genres, err := GetUsersFavouriteGenres(userId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		err = godotenv.Load(".env")
		if err != nil {
			log.Println("WARNING .env file not found!")
		}
		var recommendedMovieLimitVal int64 = 5
		recommendedMovieLimitValStr := os.Getenv("RECOMMEND_MOVIE_LIMIT")
		if recommendedMovieLimitValStr != "" {
			recommendedMovieLimitVal,_ = strconv.ParseInt(recommendedMovieLimitValStr,10,64)
		}

		findOptions := options.Find()
		findOptions.SetSort(bson.D{{Key:"ranking.ranking_value",Value:-1}})
		findOptions.SetLimit(recommendedMovieLimitVal)
		filter := bson.M{"genre.genre_name":bson.M{"$in":favourite_genres}}

		var ctx,cancel = context.WithTimeout(context.Background(),100*time.Second)
		defer cancel() 

		cursor, err := movieCollection.Find(ctx,filter,findOptions)

		if err != nil {
			c.JSON(http.StatusInternalServerError,gin.H{"error":"Error Fetching  recommended movies!"})
			return 
		}
		defer cursor.Close(ctx)

		var recommendedMovies []models.Movie
		if err := cursor.All(ctx,&recommendedMovies); err != nil {
			c.JSON(http.StatusInternalServerError,gin.H{"error":err.Error()})
			return 
		} 
		c.JSON(http.StatusOK,recommendedMovies)
	}
} 

func GetUsersFavouriteGenres(userId string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	filter := bson.M{"user_id": userId}
	projection := bson.M{
		"favourite_genres.genre_name": 1,
		"_id":                         0,
	}
	opts := options.FindOne().SetProjection(projection)
	var result bson.M
	// call
	err := userCollection.FindOne(ctx, filter, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []string{}, nil
		}
		return nil, err
	}

	favGenresArray, ok := result["favourite_genres"].(bson.A)

	if !ok {
		return []string{}, errors.New("unable to revitreve favourite genres for user")
	}
	// extracting the genre_name from our db
	// and added them into an array and return the array to the calling func
	var genreNames []string

	for _, item := range favGenresArray {
		if genreMap, ok := item.(bson.D); ok {
			for _, elem := range genreMap {
				if elem.Key == "genre_name" {
					if name, ok := elem.Value.(string); ok {
						genreNames = append(genreNames, name)
					}
				}
			}
		}
	}
	return genreNames, nil
}
