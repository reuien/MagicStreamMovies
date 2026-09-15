package controllers

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestParseMovieQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/movies?page=2&page_size=25&search=star.*&genre=Sci-Fi&sort=title_asc", nil)
	query, err := parseMovieQuery(ctx)
	if err != nil {
		t.Fatalf("parseMovieQuery() error = %v", err)
	}
	if query.Page != 2 || query.PageSize != 25 || query.Sort != "title_asc" {
		t.Fatalf("parseMovieQuery() = %#v", query)
	}
	filter := movieQueryFilter(query)
	want := bson.M{
		"title":            bson.M{"$regex": `star\.\*`, "$options": "i"},
		"genre.genre_name": bson.M{"$regex": `^Sci-Fi$`, "$options": "i"},
	}
	if !reflect.DeepEqual(filter, want) {
		t.Fatalf("movieQueryFilter() = %#v, want %#v", filter, want)
	}
}

func TestParseMovieQueryRejectsInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, target := range []string{"/movies?page=0", "/movies?page_size=101", "/movies?sort=unknown"} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("GET", target, nil)
		if _, err := parseMovieQuery(ctx); err == nil {
			t.Fatalf("parseMovieQuery(%q) accepted invalid input", target)
		}
	}
}

func TestMovieUpdateDocumentOnlyContainsProvidedFields(t *testing.T) {
	title := "Updated title"
	got := movieUpdateDocument(models.MovieUpdateRequest{Title: &title})
	want := bson.M{"title": title}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("movieUpdateDocument() = %#v, want %#v", got, want)
	}
}
