package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrConversationNotFound = errors.New("conversation not found")

type ConversationService struct {
	conversations *mongo.Collection
	feedback      *mongo.Collection
}

func NewConversationService(conversations, feedback *mongo.Collection) *ConversationService {
	return &ConversationService{conversations: conversations, feedback: feedback}
}

func (s *ConversationService) Context(ctx context.Context, userID, conversationID string) ([]string, error) {
	if conversationID == "" {
		return nil, nil
	}
	conversation, err := s.Get(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	start := 0
	if len(conversation.Messages) > 6 {
		start = len(conversation.Messages) - 6
	}
	result := make([]string, 0, len(conversation.Messages)-start)
	for _, message := range conversation.Messages[start:] {
		result = append(result, message.Role+": "+message.Content)
	}
	return result, nil
}

func (s *ConversationService) SaveTurn(ctx context.Context, userID, conversationID, query string, response models.RecommendationResponse) (string, error) {
	now := time.Now().UTC()
	movieIDs := make([]string, 0, len(response.Recommendations))
	for _, item := range response.Recommendations {
		movieIDs = append(movieIDs, item.Movie.ImdbID)
	}
	messages := []models.ConversationMessage{
		{Role: "user", Content: query, CreatedAt: now},
		{Role: "assistant", Content: recommendationSummary(response), RecommendedMovies: movieIDs, CreatedAt: now},
	}
	if conversationID == "" {
		title := []rune(strings.TrimSpace(query))
		if len(title) > 30 {
			title = title[:30]
		}
		conversation := models.Conversation{UserID: userID, Title: string(title), Messages: messages, CreatedAt: now, UpdatedAt: now}
		result, err := s.conversations.InsertOne(ctx, conversation)
		if err != nil {
			return "", fmt.Errorf("create conversation: %w", err)
		}
		id, ok := result.InsertedID.(bson.ObjectID)
		if !ok {
			return "", errors.New("invalid conversation id")
		}
		return id.Hex(), nil
	}
	id, err := bson.ObjectIDFromHex(conversationID)
	if err != nil {
		return "", ErrConversationNotFound
	}
	result, err := s.conversations.UpdateOne(ctx, bson.M{"_id": id, "user_id": userID}, bson.M{
		"$push": bson.M{"messages": bson.M{"$each": messages}},
		"$set":  bson.M{"updated_at": now},
	})
	if err != nil {
		return "", fmt.Errorf("update conversation: %w", err)
	}
	if result.MatchedCount == 0 {
		return "", ErrConversationNotFound
	}
	return conversationID, nil
}

func (s *ConversationService) List(ctx context.Context, userID string) ([]models.Conversation, error) {
	cursor, err := s.conversations.Find(ctx, bson.M{"user_id": userID}, options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetLimit(50))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var conversations []models.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}
	return conversations, nil
}

func (s *ConversationService) Get(ctx context.Context, userID, conversationID string) (models.Conversation, error) {
	id, err := bson.ObjectIDFromHex(conversationID)
	if err != nil {
		return models.Conversation{}, ErrConversationNotFound
	}
	var conversation models.Conversation
	if err := s.conversations.FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&conversation); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.Conversation{}, ErrConversationNotFound
		}
		return models.Conversation{}, err
	}
	return conversation, nil
}

func (s *ConversationService) SaveFeedback(ctx context.Context, userID, imdbID, feedbackType string) error {
	filter := bson.M{"user_id": userID, "imdb_id": imdbID}
	update := bson.M{"$set": bson.M{
		"type": feedbackType, "updated_at": time.Now().UTC(),
	}}
	_, err := s.feedback.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		_, err = s.feedback.UpdateOne(ctx, filter, update)
	}
	return err
}

func (s *ConversationService) DislikedMovieIDs(ctx context.Context, userID string) ([]string, error) {
	cursor, err := s.feedback.Find(ctx, bson.M{"user_id": userID, "type": "dislike"})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []models.RecommendationFeedback
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ImdbID)
	}
	return ids, nil
}

func recommendationSummary(response models.RecommendationResponse) string {
	if len(response.Recommendations) == 0 {
		return "没有找到符合当前条件的电影"
	}
	titles := make([]string, 0, len(response.Recommendations))
	for _, item := range response.Recommendations {
		titles = append(titles, item.Movie.Title)
	}
	return "推荐：" + strings.Join(titles, "、")
}
