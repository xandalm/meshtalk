package meshtalk_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"meshtalk"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type StubStorage struct {
	posts     map[string]meshtalk.Post
	comments  map[string]map[string]meshtalk.Comment
	editCalls []string
}

func NewStubStorage() *StubStorage {
	return &StubStorage{
		map[string]meshtalk.Post{},
		map[string]map[string]meshtalk.Comment{},
		[]string{},
	}
}

func (s *StubStorage) GetPost(id string) *meshtalk.Post {
	found, ok := s.posts[id]
	if !ok {
		return nil
	}
	return &meshtalk.Post{
		Id:        found.Id,
		Title:     found.Title,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}
}

func (s *StubStorage) GetPosts() []meshtalk.Post {
	posts := make([]meshtalk.Post, 0, len(s.posts))
	for _, post := range s.posts {
		posts = append(posts, post)
	}
	return posts
}

func (s *StubStorage) StorePost(post *meshtalk.Post) error {
	post.Id = strconv.Itoa(len(s.posts) + 1)
	createdAt := time.Now()
	post.CreatedAt = &createdAt
	s.posts[post.Id] = *post
	return nil
}

func (s *StubStorage) EditPost(post *meshtalk.Post) error {
	_, ok := s.posts[post.Id]
	if !ok {
		return meshtalk.ErrPostNotFound
	}
	s.editCalls = append(s.editCalls, post.Id)
	return nil
}

func (s *StubStorage) DeletePost(id string) error {
	delete(s.posts, id)
	return nil
}

func (s *StubStorage) GetComments(post string) []meshtalk.Comment {
	var res []meshtalk.Comment
	var do func(*[]meshtalk.Comment, *meshtalk.Comment)

	if post != "" {
		do = func(res *[]meshtalk.Comment, c *meshtalk.Comment) {
			if post == c.Post {
				*res = append(*res, *c)
			}
		}
	} else {
		do = func(res *[]meshtalk.Comment, c *meshtalk.Comment) {
			*res = append(*res, *c)
		}
	}

	for _, comments := range s.comments {
		for _, comment := range comments {
			do(&res, &comment)
		}
	}

	return res
}

func (s *StubStorage) GetComment(post, id string) *meshtalk.Comment {
	var found meshtalk.Comment
	found, ok := s.comments[post][id]
	if !ok {
		return nil
	}
	return &meshtalk.Comment{
		Id:        found.Id,
		Post:      found.Post,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}
}

type StubFailingStorage struct {
	posts map[string]meshtalk.Post
}

func (s *StubFailingStorage) GetPost(id string) *meshtalk.Post {
	return nil
}

func (s *StubFailingStorage) GetPosts() []meshtalk.Post {
	return nil
}

func (s *StubFailingStorage) StorePost(post *meshtalk.Post) error {
	return errors.New("some error")
}

func (s *StubFailingStorage) EditPost(post *meshtalk.Post) error {
	return errors.New("some error")
}

func (s *StubFailingStorage) DeletePost(id string) error {
	return errors.New("some error")
}

func (s *StubFailingStorage) GetComments(post string) []meshtalk.Comment {
	return nil
}

func (s *StubFailingStorage) GetComment(post, id string) *meshtalk.Comment {
	return nil
}

type MockStorage struct {
	GetPostFunc     func(id string) *meshtalk.Post
	GetPostsFunc    func() []meshtalk.Post
	StorePostFunc   func(post *meshtalk.Post) error
	EditPostFunc    func(post *meshtalk.Post) error
	DeletePostFunc  func(id string) error
	GetCommentsFunc func(post string) []meshtalk.Comment
	GetCommentFunc  func(post, id string) *meshtalk.Comment
}

func (s *MockStorage) GetPost(id string) *meshtalk.Post {
	return s.GetPostFunc(id)
}

func (s *MockStorage) GetPosts() []meshtalk.Post {
	return s.GetPostsFunc()
}

func (s *MockStorage) StorePost(post *meshtalk.Post) error {
	return s.StorePostFunc(post)
}

func (s *MockStorage) EditPost(post *meshtalk.Post) error {
	return s.EditPostFunc(post)
}

func (s *MockStorage) DeletePost(id string) error {
	return s.DeletePostFunc(id)
}

func (s *MockStorage) GetComments(post string) []meshtalk.Comment {
	return s.GetCommentsFunc(post)
}

func (s *MockStorage) GetComment(post, id string) *meshtalk.Comment {
	return s.GetCommentFunc(post, id)
}

func TestGetPost(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]meshtalk.Post{
			"1": {
				Id:        "1",
				Title:     "Post 1",
				Content:   "Post Content",
				Author:    "Alex",
				CreatedAt: newDate(2023, time.December, 4, 16, 30, 30, 100),
			},
			"2": {
				Id:        "2",
				Title:     "Post 2",
				Content:   "Post Content",
				Author:    "Andre",
				CreatedAt: newDate(2023, time.December, 4, 17, 0, 0, 0),
			},
		},
	}
	server := meshtalk.NewServer(storage)

	t.Run("returns post with id equal to 1", func(t *testing.T) {

		request := newGetPostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		got := getPostFromResponseModel(t, response.Body)
		want := storage.posts["1"]

		assertGotPost(t, got, want)
	})

	t.Run("returns post with id equal to 2", func(t *testing.T) {

		request := newGetPostRequest("2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		got := getPostFromResponseModel(t, response.Body)
		want := storage.posts["2"]

		assertGotPost(t, got, want)
	})

	t.Run("returns 404 on nonexistent post", func(t *testing.T) {
		request := newGetPostRequest("0")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

		got := getErrorFromResponseModel(t, response.Body)
		want := meshtalk.Error{
			meshtalk.ErrPostNotFoundMessage,
		}

		assertGotError(t, got, want)
	})

	t.Run("returns all posts", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/posts", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		responseModel := getResponseModelFromResponse(t, response.Body)

		var got []meshtalk.Post
		data, _ := json.Marshal(responseModel.Data)

		if err := json.NewDecoder(bytes.NewReader(data)).Decode(&got); err != nil {
			t.Fatalf("unable to parse data into posts list, %v", err)
		}

		for _, p := range storage.posts {
			assertContains(t, got, p)
		}

	})
}

func TestStorePost(t *testing.T) {
	storage := NewStubStorage()
	server := meshtalk.NewServer(storage)

	t.Run(`returns 201 and id equal to "1" on storage`, func(t *testing.T) {
		request := newStorePostRequest(`{"title": "Post X", "content": "Post Content", "author": "Alex"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusCreated)

		got := getPostFromResponseModel(t, response.Body)

		want := meshtalk.Post{
			Id:      "1",
			Title:   "Post X",
			Content: "Post Content",
			Author:  "Alex",
		}

		if len(storage.posts) != 1 {
			t.Errorf("expected posts list size to be %d, but got  %d", 1, len(storage.posts))
		}

		if got.Id != want.Id || got.Title != want.Title || got.Content != want.Content || got.Author != want.Author {
			t.Errorf(
				`did not get expected post, got {Id="%s", Title="%s", Content="%s", Author="%s"} want {Id="%s", Title="%s", Content="%s", Author="%s"}`,
				got.Id,
				got.Title,
				got.Content,
				got.Author,
				want.Id,
				want.Title,
				want.Content,
				want.Author,
			)
		}

		if got.CreatedAt == nil {
			t.Errorf("did not get the created datetime, got %q", got.CreatedAt)
		}

	})

	t.Run("returns 400 when request with incompatible json data", func(t *testing.T) {

		t.Run("returns 400 and unsupported error", func(t *testing.T) {
			request := newStorePostRequest(`data`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := meshtalk.Error{
				meshtalk.ErrUnsupportedPostMessage,
			}

			assertGotError(t, got, want)
		})
		t.Run("returns 400 and missing fields error", func(t *testing.T) {
			request := newStorePostRequest(`{}`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := meshtalk.Error{
				meshtalk.ErrMissingPostFieldsMessage,
			}

			assertGotError(t, got, want)
		})
	})

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &StubFailingStorage{}
		server := meshtalk.NewServer(storage)

		request := newStorePostRequest(`{"title": "Post X", "content": "Post Content", "author": "Alex"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)

	})
}

func TestEditPost(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]meshtalk.Post{
			"1": *meshtalk.NewPost("1", "Post 1", "Post Content", "Alex"),
			"2": *meshtalk.NewPost("2", "Post 2", "Post Content", "Andre"),
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
	t.Run("returns 404 on nonexistent post", func(t *testing.T) {
		jsonRaw := `{"Content": "Edited Content"}`
		request := newEditPostRequest("3", jsonRaw)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

		got := getErrorFromResponseModel(t, response.Body)
		want := meshtalk.Error{
			meshtalk.ErrPostNotFoundMessage,
		}

		assertGotError(t, got, want)
	})
	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &StubFailingStorage{
			posts: map[string]meshtalk.Post{
				"1": *meshtalk.NewPost("1", "Post 1", "Post Content", "Alex"),
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
			posts: map[string]meshtalk.Post{
				"1": *meshtalk.NewPost("1", "Post 1", "Post Content", "Alex"),
			},
		}
		server := meshtalk.NewServer(storage)
		request := newDeletePostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)
		if len(storage.posts) != 0 {
			t.Errorf("expected that the post was deleted, but it was not")
		}
	})
	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &StubFailingStorage{}
		server := meshtalk.NewServer(storage)

		request := newDeletePostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestGetComments(t *testing.T) {
	storage := &StubStorage{
		comments: map[string]map[string]meshtalk.Comment{
			"1": {
				"1": {
					Id:        "1",
					Post:      "1",
					Content:   "Some comment",
					Author:    "Alexandre",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
				"2": {
					Id:        "2",
					Post:      "1",
					Content:   "Some comment",
					Author:    "João",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
			},
			"2": {
				"1": {
					Id:        "1",
					Post:      "2",
					Content:   "Some comment",
					Author:    "Maria",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
			},
		},
	}
	server := meshtalk.NewServer(storage)

	t.Run("returns comments from post 1", func(t *testing.T) {

		t.Run("/comments?post=1", func(t *testing.T) {
			request := newCommentsRequest("1", "")
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusOK)

			got := getCommentsListFromResponseModel(t, response.Body)

			for _, c := range storage.comments["1"] {
				assertContains(t, got, c)
			}

			if len(got) != len(storage.comments["1"]) {
				t.Error("got unexpected comment(s)")
			}
		})

		t.Run("/posts/1/comments", func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodGet, "/posts/1/comments", nil)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusOK)

			got := getCommentsListFromResponseModel(t, response.Body)

			for _, c := range storage.comments["1"] {
				assertContains(t, got, c)
			}

			if len(got) != len(storage.comments["1"]) {
				t.Error("got unexpected comment(s)")
			}
		})
	})

	t.Run("returns comment 2 from post 1", func(t *testing.T) {

		t.Run("/comments?post=1&id=2", func(t *testing.T) {
			request := newCommentsRequest("1", "2")
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusOK)

			got := getCommentsListFromResponseModel(t, response.Body)

			if len(got) > 1 {
				t.Fatal("expect only one comment, but got more than one")
			}

			assertContains(t, got, storage.comments["1"]["2"])
		})

		t.Run("/posts/1/comments/2", func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodGet, "/posts/1/comments/2", nil)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusOK)

			got := getCommentFromResponseModel(t, response.Body)
			want := storage.comments["1"]["2"]

			if !reflect.DeepEqual(got, want) {
				t.Errorf("got comment %v, but want %v", got, want)
			}
		})
	})

	t.Run("returns all comments", func(t *testing.T) {
		request := newCommentsRequest("", "")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		got := getCommentsListFromResponseModel(t, response.Body)

		for _, cs := range storage.comments {
			for _, c := range cs {
				assertContains(t, got, c)
			}
		}
	})
}

func TestServerTimeout(t *testing.T) {
	t.Run("returns 408 when reaches server timeout", func(t *testing.T) {
		storage := &MockStorage{
			GetPostFunc: func(id string) *meshtalk.Post {
				time.Sleep(time.Second * 2)
				return &meshtalk.Post{}
			},
		}
		server := meshtalk.NewServer(storage)
		server.SetTimeout(time.Second * 1)

		request := newGetPostRequest("1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusRequestTimeout)
	})
}

func newDate(year int, month time.Month, day, hour, min, sec, mlsec int) *time.Time {
	datetime := time.Date(year, month, day, hour, min, sec, mlsec*1e6, time.UTC)
	return &datetime
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

func newCommentsRequest(postId, commentId string) *http.Request {
	url := "/comments"
	if postId != "" {
		url = url + "?post=" + postId
		if commentId != "" {
			url = url + "&id=" + commentId
		}
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	return req
}

func assertStatus(t testing.TB, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("did not get correct status, got %d but want %d", response.Code, want)
	}
}

func assertGotPost(t testing.TB, got, want meshtalk.Post) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong post received, got %q but want %q", got, want)
	}
}

func assertGotError(t testing.TB, got, want meshtalk.Error) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got error %q, but want %q", got, want)
	}
}

func assertContains[T any](t testing.TB, list []T, needle T) {
	t.Helper()
	contains := false
	for _, n := range list {
		if reflect.DeepEqual(n, needle) {
			contains = true
			break
		}
	}
	if !contains {
		t.Errorf("expected %v to contain %v but it didn't", list, needle)
	}
}

func getResponseModelFromResponse(t *testing.T, body io.Reader) meshtalk.ResponseModel {
	t.Helper()

	var responseModel meshtalk.ResponseModel
	err := json.NewDecoder(body).Decode(&responseModel)
	if err != nil {
		t.Fatalf("unable to parse response from server into ResponseModel, %v", err)
	}

	return responseModel
}

func getPostFromResponseModel(t *testing.T, body io.Reader) meshtalk.Post {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)

	var post meshtalk.Post
	data, _ := json.Marshal(responseModel.Data)

	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&post); err != nil {
		t.Fatalf("unable to parse data from ResponseModel into Post, %v", err)
	}

	return post
}

func getErrorFromResponseModel(t *testing.T, body io.Reader) meshtalk.Error {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)

	var responseError meshtalk.Error
	data, _ := json.Marshal(responseModel.Error)

	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&responseError); err != nil {
		t.Fatalf("unable to parse error from ResponseModel into Error, %v", err)
	}

	return responseError
}

func getCommentsListFromResponseModel(t *testing.T, body io.Reader) []meshtalk.Comment {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)
	data, _ := json.Marshal(responseModel.Data)

	var list []meshtalk.Comment
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&list); err != nil {
		t.Fatalf("unable to parse data into comments list, %v", err)
	}

	return list
}

func getCommentFromResponseModel(t *testing.T, body io.Reader) meshtalk.Comment {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)
	data, _ := json.Marshal(responseModel.Data)

	var c meshtalk.Comment
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&c); err != nil {
		t.Fatalf("unable to parse data into comments list, %v", err)
	}

	return c
}
