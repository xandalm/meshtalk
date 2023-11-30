package meshtalk_test

import (
	"encoding/json"
	"meshtalk"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type StubStorage struct {
	posts map[string]*meshtalk.Post
}

func (p *StubStorage) GetPost(id string) *meshtalk.Post {
	return p.posts[id]
}

func (p *StubStorage) StorePost(post *meshtalk.Post) string {
	id := len(p.posts) + 1
	return strconv.Itoa(id)
}

func TestGetPost(t *testing.T) {
	storage := &StubStorage{
		map[string]*meshtalk.Post{
			"1": meshtalk.NewPost("1", "Post 1", "Post Content"),
			"2": meshtalk.NewPost("2", "Post 2", "Post Content"),
		},
	}
	server := meshtalk.NewServer(storage)

	t.Run("returns post with id equal to 1", func(t *testing.T) {

		request := newGetPostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		var got meshtalk.Post
		json.NewDecoder(response.Body).Decode(&got)

		assertStatus(t, response.Code, http.StatusOK)
		assertGotPost(t, &got, storage.posts["1"])
	})

	t.Run("returns post with id equal to 2", func(t *testing.T) {

		request := newGetPostRequest("2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		var got meshtalk.Post
		json.NewDecoder(response.Body).Decode(&got)

		assertStatus(t, response.Code, http.StatusOK)
		assertGotPost(t, &got, storage.posts["2"])
	})

	t.Run("returns 404 on missing posts", func(t *testing.T) {
		request := newGetPostRequest("0")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response.Code, http.StatusNotFound)
	})
}

func TestStorePost(t *testing.T) {
	storage := &StubStorage{}
	server := meshtalk.NewServer(storage)

	t.Run("returns created on POST", func(t *testing.T) {
		request := newStorePostRequest(`{
"Title": "Post X",
"Content": "Post Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response.Code, http.StatusCreated)

		got := response.Body.String()
		want := "1"

		if got != want {
			t.Errorf("did not get expected id, got %q want %q", got, want)
		}
	})

	t.Run("returns 400 when request with incompatible json data", func(t *testing.T) {

		request := newStorePostRequest(`x`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response.Code, http.StatusBadRequest)
	})
}

func newGetPostRequest(id string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/posts/"+id, nil)
	return req
}

func newStorePostRequest(jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/posts", strings.NewReader(jsonRaw))
	return req
}

func assertStatus(t testing.TB, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("did not get correct status, got %d but want %d", got, want)
	}
}

func assertGotPost(t testing.TB, got, want *meshtalk.Post) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong post received, got %q but want %q", got, want)
	}
}
