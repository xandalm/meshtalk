package post

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPost(t *testing.T) {

	t.Run("returns post with id equal to 1", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/posts/1", nil)
		response := httptest.NewRecorder()

		GetPost(response, request)

		got := response.Body.String()
		want := `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("returns post with id equal to 2", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/posts/2", nil)
		response := httptest.NewRecorder()

		GetPost(response, request)

		got := response.Body.String()
		want := `{"ID": "2", "Title": "Post 2", "Content": "Post Content"}`

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
