package post

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPost(t *testing.T) {

	t.Run("returns post with id equal to 1", func(t *testing.T) {
		request := newPostRequest("1")
		response := httptest.NewRecorder()

		PostServer(response, request)

		got := response.Body.String()
		want := `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`

		assertGotPost(t, got, want)
	})
	t.Run("returns post with id equal to 2", func(t *testing.T) {
		request := newPostRequest("2")
		response := httptest.NewRecorder()

		PostServer(response, request)

		got := response.Body.String()
		want := `{"ID": "2", "Title": "Post 2", "Content": "Post Content"}`

		assertGotPost(t, got, want)
	})
}

func newPostRequest(id string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/posts/"+id, nil)
	return req
}

func assertGotPost(t testing.TB, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("wrong post received, got %q but want %q", got, want)
	}
}
