package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"github.com/tmc/langchaingo/llms/openai"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const preferencePrompt = `You extract movie preferences from a user's message.
Return JSON only, without markdown or commentary, using this schema:
{"genres":[],"excluded_genres":[],"keywords":[],"mood":"","max_results":5}
Use short genre names. max_results must be between 1 and 10.
User message: `

type TextGenerator interface {
	Generate(context.Context, string) (string, error)
}

type LangChainGenerator struct {
	llm *openai.LLM
}

func NewLangChainGenerator() (*LangChainGenerator, error) {
	return NewLangChainGeneratorWithConfig(os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_MODEL"))
}

func NewLangChainGeneratorWithConfig(apiKey, baseURL, model string) (*LangChainGenerator, error) {
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}
	opts := []openai.Option{openai.WithToken(apiKey)}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	if model != "" {
		opts = append(opts, openai.WithModel(model))
	}
	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create AI client: %w", err)
	}
	return &LangChainGenerator{llm: llm}, nil
}

func (g *LangChainGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	return g.llm.Call(ctx, prompt)
}

type RecommendationService struct {
	movies    *mongo.Collection
	audits    *mongo.Collection
	generator TextGenerator
}

func NewRecommendationService(movies *mongo.Collection, generator TextGenerator, audits ...*mongo.Collection) *RecommendationService {
	service := &RecommendationService{movies: movies, generator: generator}
	if len(audits) > 0 {
		service.audits = audits[0]
	}
	return service
}

func ParseMoviePreferences(raw string) (models.MoviePreferences, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var preferences models.MoviePreferences
	decoder := json.NewDecoder(strings.NewReader(cleaned))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&preferences); err != nil {
		return preferences, fmt.Errorf("parse AI preferences: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return preferences, errors.New("parse AI preferences: trailing JSON data")
	}
	preferences.Genres = normalizeTerms(preferences.Genres)
	preferences.ExcludedGenres = normalizeTerms(preferences.ExcludedGenres)
	preferences.Keywords = normalizeTerms(preferences.Keywords)
	preferences.Mood = strings.TrimSpace(preferences.Mood)
	if preferences.MaxResults < 1 || preferences.MaxResults > 10 {
		preferences.MaxResults = 5
	}
	if err := validatePreferences(preferences); err != nil {
		return preferences, err
	}
	return preferences, nil
}

func validatePreferences(preferences models.MoviePreferences) error {
	if len(preferences.Genres) > 10 || len(preferences.ExcludedGenres) > 10 || len(preferences.Keywords) > 10 {
		return errors.New("AI preferences contain too many values")
	}
	for _, values := range [][]string{preferences.Genres, preferences.ExcludedGenres, preferences.Keywords} {
		for _, value := range values {
			if len([]rune(value)) > 50 {
				return errors.New("AI preference value is too long")
			}
		}
	}
	if len([]rune(preferences.Mood)) > 100 {
		return errors.New("AI mood value is too long")
	}
	return nil
}

func normalizeTerms(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *RecommendationService) Recommend(ctx context.Context, userID, query string, history []string, excludedMovieIDs []string) (response models.RecommendationResponse, resultErr error) {
	startedAt := time.Now()
	attempts := 0
	defer func() {
		s.audit(userID, query, attempts, startedAt, resultErr)
	}()
	query = strings.TrimSpace(query)
	if query == "" {
		return models.RecommendationResponse{}, errors.New("recommendation query is empty")
	}
	preferences, attempts, err := s.extractPreferences(ctx, query, history)
	if err != nil {
		return models.RecommendationResponse{}, err
	}

	filter := buildMovieFilter(preferences, excludedMovieIDs)
	findOptions := options.Find().SetSort(bson.D{{Key: "ranking.ranking_value", Value: -1}}).SetLimit(preferences.MaxResults)
	cursor, err := s.movies.Find(ctx, filter, findOptions)
	if err != nil {
		return models.RecommendationResponse{}, fmt.Errorf("find recommendation candidates: %w", err)
	}
	defer cursor.Close(ctx)

	var movies []models.Movie
	if err := cursor.All(ctx, &movies); err != nil {
		return models.RecommendationResponse{}, fmt.Errorf("decode recommendation candidates: %w", err)
	}
	items := make([]models.MovieRecommendation, 0, len(movies))
	for _, movie := range movies {
		items = append(items, models.MovieRecommendation{Movie: movie, Score: movie.Ranking.RankingValue, Reason: BuildRecommendationReason(movie, preferences)})
	}
	return models.RecommendationResponse{Query: query, Preferences: preferences, Recommendations: items}, nil
}

func (s *RecommendationService) extractPreferences(ctx context.Context, query string, history []string) (models.MoviePreferences, int, error) {
	prompt := preferencePrompt + query
	if len(history) > 0 {
		prompt += "\nRecent conversation context: " + strings.Join(history, " | ")
	}
	raw, err := s.generator.Generate(ctx, prompt)
	if err != nil {
		return models.MoviePreferences{}, 1, fmt.Errorf("extract movie preferences: %w", err)
	}
	preferences, err := ParseMoviePreferences(raw)
	if err == nil {
		return preferences, 1, nil
	}
	if len(raw) > 4000 {
		raw = raw[:4000]
	}
	repairPrompt := `Repair the following output into exactly one JSON object with only these fields: genres, excluded_genres, keywords, mood, max_results. Return JSON only. Output: ` + raw
	repaired, repairErr := s.generator.Generate(ctx, repairPrompt)
	if repairErr != nil {
		return models.MoviePreferences{}, 2, fmt.Errorf("repair AI preferences: %w", repairErr)
	}
	preferences, err = ParseMoviePreferences(repaired)
	if err != nil {
		return models.MoviePreferences{}, 2, fmt.Errorf("AI preferences invalid after repair: %w", err)
	}
	return preferences, 2, nil
}

func (s *RecommendationService) audit(userID, query string, attempts int, startedAt time.Time, resultErr error) {
	if s.audits == nil {
		return
	}
	status, errorCode := "success", ""
	if resultErr != nil {
		status, errorCode = "failed", "recommendation_failed"
	}
	hash := sha256.Sum256([]byte(query))
	audit := models.AIInvocationAudit{
		UserID: userID, QueryHash: fmt.Sprintf("%x", hash), Model: os.Getenv("OPENAI_MODEL"), Status: status,
		Attempts: attempts, DurationMS: time.Since(startedAt).Milliseconds(), ErrorCode: errorCode, CreatedAt: time.Now().UTC(),
	}
	auditCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = s.audits.InsertOne(auditCtx, audit)
}

func buildMovieFilter(preferences models.MoviePreferences, excludedMovieIDs []string) bson.M {
	conditions := bson.A{}
	if len(preferences.Genres) > 0 {
		conditions = append(conditions, bson.M{"genre.genre_name": bson.M{"$in": preferences.Genres}})
	}
	if len(preferences.ExcludedGenres) > 0 {
		conditions = append(conditions, bson.M{"genre.genre_name": bson.M{"$nin": preferences.ExcludedGenres}})
	}
	if len(excludedMovieIDs) > 0 {
		conditions = append(conditions, bson.M{"imdb_id": bson.M{"$nin": excludedMovieIDs}})
	}
	if len(preferences.Keywords) > 0 {
		keywordConditions := bson.A{}
		for _, keyword := range preferences.Keywords {
			safeKeyword := regexp.QuoteMeta(keyword)
			keywordConditions = append(keywordConditions,
				bson.M{"title": bson.M{"$regex": safeKeyword, "$options": "i"}},
				bson.M{"admin_review": bson.M{"$regex": safeKeyword, "$options": "i"}},
			)
		}
		conditions = append(conditions, bson.M{"$or": keywordConditions})
	}
	if len(conditions) == 0 {
		return bson.M{}
	}
	return bson.M{"$and": conditions}
}

func BuildRecommendationReason(movie models.Movie, preferences models.MoviePreferences) string {
	matched := make([]string, 0)
	wanted := make(map[string]struct{}, len(preferences.Genres))
	for _, genre := range preferences.Genres {
		wanted[strings.ToLower(genre)] = struct{}{}
	}
	for _, genre := range movie.Genre {
		if _, ok := wanted[strings.ToLower(genre.GenreName)]; ok {
			matched = append(matched, genre.GenreName)
		}
	}
	parts := make([]string, 0, 2)
	if len(matched) > 0 {
		parts = append(parts, "匹配你想看的"+strings.Join(matched, "、")+"类型")
	}
	if preferences.Mood != "" {
		parts = append(parts, "符合“"+preferences.Mood+"”的观影氛围")
	}
	if len(parts) == 0 {
		return "根据影片评分和你的描述筛选出的候选影片"
	}
	return strings.Join(parts, "，")
}
