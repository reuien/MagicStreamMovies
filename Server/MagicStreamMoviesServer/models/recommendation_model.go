package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type RecommendationRequest struct {
	Message        string `json:"message" binding:"required,min=2,max=1000"`
	ConversationID string `json:"conversation_id,omitempty"`
}

type MoviePreferences struct {
	Genres         []string `json:"genres"`
	ExcludedGenres []string `json:"excluded_genres"`
	Keywords       []string `json:"keywords"`
	Mood           string   `json:"mood"`
	MaxResults     int64    `json:"max_results"`
}

type MovieRecommendation struct {
	Movie  Movie  `json:"movie"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

type RecommendationResponse struct {
	ConversationID  string                `json:"conversation_id"`
	Query           string                `json:"query"`
	Preferences     MoviePreferences      `json:"preferences"`
	Recommendations []MovieRecommendation `json:"recommendations"`
}

type ConversationMessage struct {
	Role              string    `bson:"role" json:"role"`
	Content           string    `bson:"content" json:"content"`
	RecommendedMovies []string  `bson:"recommended_movies,omitempty" json:"recommended_movies,omitempty"`
	CreatedAt         time.Time `bson:"created_at" json:"created_at"`
}

type Conversation struct {
	ID        bson.ObjectID         `bson:"_id,omitempty" json:"id"`
	UserID    string                `bson:"user_id" json:"user_id"`
	Title     string                `bson:"title" json:"title"`
	Messages  []ConversationMessage `bson:"messages" json:"messages"`
	CreatedAt time.Time             `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time             `bson:"updated_at" json:"updated_at"`
}

type RecommendationFeedbackRequest struct {
	Type string `json:"type" binding:"required,oneof=like dislike"`
}

type RecommendationFeedback struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string        `bson:"user_id" json:"user_id"`
	ImdbID    string        `bson:"imdb_id" json:"imdb_id"`
	Type      string        `bson:"type" json:"type"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}
