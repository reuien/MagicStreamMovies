package models

type RecommendationRequest struct {
	Message string `json:"message" binding:"required,min=2,max=1000"`
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
	Query           string                `json:"query"`
	Preferences     MoviePreferences      `json:"preferences"`
	Recommendations []MovieRecommendation `json:"recommendations"`
}
