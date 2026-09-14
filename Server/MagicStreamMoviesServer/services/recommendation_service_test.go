package services

import (
	"reflect"
	"testing"

	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestParseMoviePreferences(t *testing.T) {
	raw := "```json\n{\"genres\":[\"科幻\",\" 科幻 \"],\"excluded_genres\":[\"恐怖\"],\"keywords\":[\"太空\"],\"mood\":\"轻松\",\"max_results\":20}\n```"
	preferences, err := ParseMoviePreferences(raw)
	if err != nil {
		t.Fatalf("ParseMoviePreferences() error = %v", err)
	}
	if len(preferences.Genres) != 1 || preferences.Genres[0] != "科幻" {
		t.Fatalf("genres = %#v, want one normalized genre", preferences.Genres)
	}
	if preferences.MaxResults != 5 {
		t.Fatalf("max_results = %d, want 5", preferences.MaxResults)
	}
}

func TestBuildMovieFilterEscapesKeywords(t *testing.T) {
	preferences := models.MoviePreferences{
		Genres:         []string{"科幻"},
		ExcludedGenres: []string{"恐怖"},
		Keywords:       []string{"星球.*"},
	}
	want := bson.M{"$and": bson.A{
		bson.M{"genre.genre_name": bson.M{"$in": []string{"科幻"}}},
		bson.M{"genre.genre_name": bson.M{"$nin": []string{"恐怖"}}},
		bson.M{"$or": bson.A{
			bson.M{"title": bson.M{"$regex": `星球\.\*`, "$options": "i"}},
			bson.M{"admin_review": bson.M{"$regex": `星球\.\*`, "$options": "i"}},
		}},
	}}
	if got := buildMovieFilter(preferences, nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("buildMovieFilter() = %#v, want %#v", got, want)
	}
}

func TestParseMoviePreferencesRejectsInvalidJSON(t *testing.T) {
	if _, err := ParseMoviePreferences("not-json"); err == nil {
		t.Fatal("ParseMoviePreferences() accepted invalid JSON")
	}
}

func TestBuildRecommendationReason(t *testing.T) {
	movie := models.Movie{Genre: []models.Genre{{GenreName: "科幻"}, {GenreName: "冒险"}}}
	preferences := models.MoviePreferences{Genres: []string{"科幻"}, Mood: "轻松"}
	reason := BuildRecommendationReason(movie, preferences)
	if reason != "匹配你想看的科幻类型，符合“轻松”的观影氛围" {
		t.Fatalf("reason = %q", reason)
	}
}

func TestBuildMovieFilterExcludesDislikedMovies(t *testing.T) {
	got := buildMovieFilter(models.MoviePreferences{}, []string{"tt-1", "tt-2"})
	want := bson.M{"$and": bson.A{bson.M{"imdb_id": bson.M{"$nin": []string{"tt-1", "tt-2"}}}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildMovieFilter() = %#v, want %#v", got, want)
	}
}
