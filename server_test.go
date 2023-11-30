package meshtalk

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type StubStorage struct {
	posts map[string]string
}

func (p *StubStorage) GetPost(id string) string {
	return p.posts[id]
}

func TestGetPost(t *testing.T) {
	storage := &StubStorage{
		map[string]string{
			"1": `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`,
			"2": `{"ID": "2", "Title": "Post 2", "Content": "Post Content"}`,
		},
	}
	server := &Server{storage}

	t.Run("returns post with id equal to 1", func(t *testing.T) {

		request := newPostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Body.String()

		assertStatus(t, response.Code, http.StatusOK)
		assertGotPost(t, got, storage.posts["1"])
	})

	t.Run("returns post with id equal to 2", func(t *testing.T) {

		request := newPostRequest("2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Body.String()

		assertStatus(t, response.Code, http.StatusOK)
		assertGotPost(t, got, storage.posts["2"])
	})

	t.Run("returns 404 on missing posts", func(t *testing.T) {
		request := newPostRequest("0")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response.Code, http.StatusNotFound)
	})
}

func newPostRequest(id string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/posts/"+id, nil)
	return req
}

func assertStatus(t testing.TB, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("did not get correct status, got %d but want %d", got, want)
	}
}

func assertGotPost(t testing.TB, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("wrong post received, got %q but want %q", got, want)
	}
}
