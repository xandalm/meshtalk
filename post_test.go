package post

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type StubPostStorage struct {
	posts map[string]string
}

func (p *StubPostStorage) GetPost(id string) string {
	return p.posts[id]
}

func TestGetPost(t *testing.T) {
	storage := &StubPostStorage{
		map[string]string{
			"1": `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`,
			"2": `{"ID": "2", "Title": "Post 2", "Content": "Post Content"}`,
		},
	}
	server := &PostServer{storage}

	t.Run("returns post with id equal to 1", func(t *testing.T) {

		request := newPostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Body.String()

		assertGotPost(t, got, storage.posts["1"])
	})

	t.Run("returns post with id equal to 2", func(t *testing.T) {

		request := newPostRequest("2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Body.String()

		assertGotPost(t, got, storage.posts["2"])
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
