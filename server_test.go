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
	posts     map[string]*meshtalk.Post
	editCalls []string
}

func NewStubStorage() *StubStorage {
	return &StubStorage{
		map[string]*meshtalk.Post{},
		[]string{},
	}
}

func (s *StubStorage) GetPost(id string) *meshtalk.Post {
	return s.posts[id]
}

func (s *StubStorage) StorePost(post *meshtalk.Post) string {
	id := len(s.posts) + 1
	return strconv.Itoa(id)
}

func (s *StubStorage) EditPost(post *meshtalk.Post) bool {
	s.editCalls = append(s.editCalls, post.Id)
	return true
}

func (s *StubStorage) DeletePost(id string) bool {
	delete(s.posts, id)
	return true
}

type StubFailingStorage struct {
	posts map[string]*meshtalk.Post
}

func (s *StubFailingStorage) GetPost(id string) *meshtalk.Post {
	return s.posts[id]
}

func (s *StubFailingStorage) StorePost(post *meshtalk.Post) string {
	return ""
}

func (s *StubFailingStorage) EditPost(post *meshtalk.Post) bool {
	return false
}

func (s *StubFailingStorage) DeletePost(id string) bool {
	return false
}

func TestGetPost(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]*meshtalk.Post{
			"1": meshtalk.NewPost("1", "Post 1", "Post Content", "Alex"),
			"2": meshtalk.NewPost("2", "Post 2", "Post Content", "Andre"),
		},
	}
	server := meshtalk.NewServer(storage)

	t.Run("returns post with id equal to 1", func(t *testing.T) {

		request := newGetPostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		var got meshtalk.Post
		json.NewDecoder(response.Body).Decode(&got)

		assertStatus(t, response, http.StatusOK)
		assertGotPost(t, &got, storage.posts["1"])
	})

	t.Run("returns post with id equal to 2", func(t *testing.T) {

		request := newGetPostRequest("2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		var got meshtalk.Post
		json.NewDecoder(response.Body).Decode(&got)

		assertStatus(t, response, http.StatusOK)
		assertGotPost(t, &got, storage.posts["2"])
	})

	t.Run("returns 404 on missing posts", func(t *testing.T) {
		request := newGetPostRequest("0")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)
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

		assertStatus(t, response, http.StatusCreated)

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

		assertStatus(t, response, http.StatusBadRequest)
	})
}

func TestEditPost(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]*meshtalk.Post{
			"1": meshtalk.NewPost("1", "Post 1", "Post Content", "Alex"),
			"2": meshtalk.NewPost("2", "Post 2", "Post Content", "Andre"),
		},
	}
	server := meshtalk.NewServer(storage)

	t.Run("returns 204 on post edited", func(t *testing.T) {
		jsonRaw := `{"Content": "Edited Content"}`
		request := newEditPostRequest("1", jsonRaw)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if len(storage.editCalls) != 1 {
			t.Error("did not edited the post")
		}
	})
	t.Run("returns 404 on missing post when edit", func(t *testing.T) {
		jsonRaw := `{"Content": "Edited Content"}`
		request := newEditPostRequest("3", jsonRaw)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)
	})
	t.Run("returns 500 on fail edit", func(t *testing.T) {
		storage := &StubFailingStorage{
			posts: map[string]*meshtalk.Post{
				"1": meshtalk.NewPost("1", "Post 1", "Post Content", "Alex"),
			},
		}
		server := meshtalk.NewServer(storage)
		jsonRaw := `{"Content": "Edited Content"}`
		request := newEditPostRequest("1", jsonRaw)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestDeletePost(t *testing.T) {
	t.Run("returns 200 on post deleted", func(t *testing.T) {
		storage := &StubStorage{
			posts: map[string]*meshtalk.Post{
				"1": meshtalk.NewPost("1", "Post 1", "Post Content", "Alex"),
				"2": meshtalk.NewPost("2", "Post 2", "Post Content", "Andre"),
			},
		}
		server := meshtalk.NewServer(storage)
		request := newDeletePostRequest("2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)
		if len(storage.posts) != 1 {
			t.Errorf("expected that the post was deleted, but it was not")
		}
	})
	t.Run("returns 500 on fail delete", func(t *testing.T) {
		storage := &StubFailingStorage{}
		server := meshtalk.NewServer(storage)

		request := newDeletePostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
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

func newEditPostRequest(id, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPut, "/posts/"+id, strings.NewReader(jsonRaw))
	return req
}

func newDeletePostRequest(id string) *http.Request {
	req, _ := http.NewRequest(http.MethodDelete, "/posts/"+id, nil)
	return req
}

func assertStatus(t testing.TB, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Errorf("did not get correct status, got %d but want %d", response.Code, want)
	}
}

func assertGotPost(t testing.TB, got, want *meshtalk.Post) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong post received, got %q but want %q", got, want)
	}
}
