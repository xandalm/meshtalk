package meshtalk_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"meshtalk"
	"meshtalk/domain/entities"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type StubStorage struct {
	posts            map[string]entities.Post
	comments         map[string]map[string]entities.Comment
	postEditCalls    []string
	commentEditCalls []string
}

func NewStubStorage() *StubStorage {
	return &StubStorage{
		map[string]entities.Post{},
		map[string]map[string]entities.Comment{},
		[]string{},
		[]string{},
	}
}

func (s *StubStorage) GetPost(id string) *entities.Post {
	found, ok := s.posts[id]
	if !ok {
		return nil
	}
	return &entities.Post{
		Id:        found.Id,
		Title:     found.Title,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}
}

func (s *StubStorage) GetPosts() []entities.Post {
	posts := make([]entities.Post, 0, len(s.posts))
	for _, post := range s.posts {
		posts = append(posts, post)
	}
	return posts
}

func (s *StubStorage) StorePost(post *entities.Post) error {
	post.Id = strconv.Itoa(len(s.posts) + 1)
	post.CreatedAt = timeToString(time.Now())
	s.posts[post.Id] = *post
	return nil
}

func timeToString(t time.Time) string {
	b, _ := t.UTC().MarshalText()
	return string(b)
}

func (s *StubStorage) EditPost(post *entities.Post) error {
	_, ok := s.posts[post.Id]
	if !ok {
		return meshtalk.ErrPostNotFound
	}
	s.postEditCalls = append(s.postEditCalls, post.Id)
	return nil
}

func (s *StubStorage) DeletePost(id string) error {
	delete(s.posts, id)
	return nil
}

func (s *StubStorage) GetComments(post string) []entities.Comment {
	var res []entities.Comment
	var do func(*[]entities.Comment, *entities.Comment)

	if post != "" {
		do = func(res *[]entities.Comment, c *entities.Comment) {
			if post == c.Post {
				*res = append(*res, *c)
			}
		}
	} else {
		do = func(res *[]entities.Comment, c *entities.Comment) {
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

func (s *StubStorage) GetComment(post, id string) *entities.Comment {
	var found entities.Comment
	found, ok := s.comments[post][id]
	if !ok {
		return nil
	}
	return &entities.Comment{
		Id:        found.Id,
		Post:      found.Post,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}
}

func (s *StubStorage) StoreComment(comment *entities.Comment) error {
	_, hasComments := s.comments[comment.Post]
	if !hasComments {
		s.comments[comment.Post] = make(map[string]entities.Comment)
	}
	comment.Id = strconv.Itoa(len(s.comments[comment.Post]) + 1)
	comment.CreatedAt = timeToString(time.Now())
	s.comments[comment.Post][comment.Id] = *comment
	return nil
}

func (s *StubStorage) EditComment(comment *entities.Comment) error {
	comments, ok := s.comments[comment.Post]
	if !ok {
		return meshtalk.ErrCommentNotFound
	}
	_, ok = comments[comment.Id]
	if !ok {
		return meshtalk.ErrCommentNotFound
	}
	s.commentEditCalls = append(s.commentEditCalls, fmt.Sprintf("%+v", comment))
	return nil
}

var errFoo = errors.New("some error")

type StubFailingStorage struct {
	posts map[string]entities.Post
}

func (s *StubFailingStorage) GetPost(id string) *entities.Post {
	return nil
}

func (s *StubFailingStorage) GetPosts() []entities.Post {
	return nil
}

func (s *StubFailingStorage) StorePost(post *entities.Post) error {
	return errFoo
}

func (s *StubFailingStorage) EditPost(post *entities.Post) error {
	return errFoo
}

func (s *StubFailingStorage) DeletePost(id string) error {
	return errFoo
}

func (s *StubFailingStorage) GetComments(post string) []entities.Comment {
	return nil
}

func (s *StubFailingStorage) GetComment(post, id string) *entities.Comment {
	return nil
}

func (s *StubFailingStorage) StoreComment(comment *entities.Comment) error {
	return errFoo
}

func (s *StubFailingStorage) EditComment(comment *entities.Comment) error {
	return errFoo
}

type MockStorage struct {
	GetPostFunc      func(id string) *entities.Post
	GetPostsFunc     func() []entities.Post
	StorePostFunc    func(post *entities.Post) error
	EditPostFunc     func(post *entities.Post) error
	DeletePostFunc   func(id string) error
	GetCommentsFunc  func(post string) []entities.Comment
	GetCommentFunc   func(post, id string) *entities.Comment
	StoreCommentFunc func(comment *entities.Comment) error
	EditCommentFunc  func(comment *entities.Comment) error
}

func (s *MockStorage) GetPost(id string) *entities.Post {
	return s.GetPostFunc(id)
}

func (s *MockStorage) GetPosts() []entities.Post {
	return s.GetPostsFunc()
}

func (s *MockStorage) StorePost(post *entities.Post) error {
	return s.StorePostFunc(post)
}

func (s *MockStorage) EditPost(post *entities.Post) error {
	return s.EditPostFunc(post)
}

func (s *MockStorage) DeletePost(id string) error {
	return s.DeletePostFunc(id)
}

func (s *MockStorage) GetComments(post string) []entities.Comment {
	return s.GetCommentsFunc(post)
}

func (s *MockStorage) GetComment(post, id string) *entities.Comment {
	return s.GetCommentFunc(post, id)
}

func (s *MockStorage) StoreComment(comment *entities.Comment) error {
	return s.StoreCommentFunc(comment)
}

func (s *MockStorage) EditComment(comment *entities.Comment) error {
	return s.EditCommentFunc(comment)
}

func TestGETPosts(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]entities.Post{
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
		want := meshtalk.ErrPostNotFound

		assertGotError(t, got, want)
	})

	t.Run("returns all posts", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/posts", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		responseModel := getResponseModelFromResponse(t, response.Body)

		var got []entities.Post
		data, _ := json.Marshal(responseModel.Data)

		if err := json.NewDecoder(bytes.NewReader(data)).Decode(&got); err != nil {
			t.Fatalf("unable to parse data into posts list, %v", err)
		}

		for _, p := range storage.posts {
			assertContains(t, got, p)
		}

	})
}

func TestPOSTPosts(t *testing.T) {
	storage := NewStubStorage()
	server := meshtalk.NewServer(storage)

	t.Run(`returns 201 and post after store post`, func(t *testing.T) {
		request := newStorePostRequest(`{"title": "Post X", "content": "Post Content", "author": "Alex"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusCreated)

		got := getPostFromResponseModel(t, response.Body)

		want := entities.Post{
			Id:      "1",
			Title:   "Post X",
			Content: "Post Content",
			Author:  "Alex",
		}

		if _, ok := storage.posts["1"]; !ok {
			t.Fatal("didn't stores the post")
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
	})

	t.Run("returns 400 when request with incompatible json data", func(t *testing.T) {

		t.Run("returns 400 and unsupported error", func(t *testing.T) {
			request := newStorePostRequest(`data`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := meshtalk.ErrUnsupportedPost

			assertGotError(t, got, want)
		})
		t.Run("returns 400 and missing fields error", func(t *testing.T) {
			request := newStorePostRequest(`{}`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := meshtalk.ErrMissingPostFields

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

func TestPUTPosts(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]entities.Post{
			"1": *entities.NewPost("1", "Post 1", "Post Content", "Alex"),
			"2": *entities.NewPost("2", "Post 2", "Post Content", "Andre"),
		},
	}
	server := meshtalk.NewServer(storage)

	t.Run("returns 204 on post edited", func(t *testing.T) {
		jsonRaw := `{"Content": "Edited Content"}`
		request := newEditPostRequest("1", jsonRaw)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if len(storage.postEditCalls) != 1 {
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
		want := meshtalk.ErrPostNotFound

		assertGotError(t, got, want)
	})
	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &StubFailingStorage{
			posts: map[string]entities.Post{
				"1": *entities.NewPost("1", "Post 1", "Post Content", "Alex"),
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

func TestDELETEPosts(t *testing.T) {
	t.Run("returns 200 on post deleted", func(t *testing.T) {
		storage := &StubStorage{
			posts: map[string]entities.Post{
				"1": *entities.NewPost("1", "Post 1", "Post Content", "Alex"),
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

func TestGETComments(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]entities.Post{
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
		comments: map[string]map[string]entities.Comment{
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

		t.Run("for /comments?post=1", func(t *testing.T) {
			request := newGetCommentsRequest("1", "")
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

		t.Run("for /posts/1/comments", func(t *testing.T) {
			request := newGetPostCommentsRequest("1", "")
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

		t.Run("for /comments?post=1&id=2", func(t *testing.T) {
			request := newGetCommentsRequest("1", "2")
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusOK)

			got := getCommentsListFromResponseModel(t, response.Body)

			if len(got) > 1 {
				t.Fatal("expect only one comment, but got more than one")
			}

			assertContains(t, got, storage.comments["1"]["2"])
		})

		t.Run("for /posts/1/comments/2", func(t *testing.T) {
			request := newGetPostCommentsRequest("1", "2")
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

	t.Run("returns 404 when try to get comments from post 3", func(t *testing.T) {
		request := newGetPostCommentsRequest("3", "")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

		got := getErrorFromResponseModel(t, response.Body)
		want := meshtalk.ErrPostNotFound

		assertGotError(t, got, want)
	})

	t.Run("returns 404 when try to get comment 3 from post 2", func(t *testing.T) {
		request := newGetPostCommentsRequest("2", "3")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

		got := getErrorFromResponseModel(t, response.Body)
		want := meshtalk.ErrCommentNotFound

		assertGotError(t, got, want)
	})

	t.Run("returns all comments", func(t *testing.T) {
		request := newGetCommentsRequest("", "")
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

func TestPOSTComments(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]entities.Post{
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
		comments: map[string]map[string]entities.Comment{},
	}
	server := meshtalk.NewServer(storage)

	t.Run(`returns 201 and comment after store comment`, func(t *testing.T) {
		request := newStoreCommentRequest("1", `{"content": "Comment Content", "author": "Alex"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusCreated)

		got := getCommentFromResponseModel(t, response.Body)
		want := entities.Comment{
			Id:      "1",
			Post:    "1",
			Content: "Comment Content",
			Author:  "Alex",
		}

		comments, ok := storage.comments["1"]
		if !ok {
			t.Fatal("didn't contains any comment into post 1")
		}

		if _, ok := comments["1"]; !ok {
			t.Fatal("didn't stores the comment")
		}

		if got.Id != want.Id || got.Post != want.Post || got.Content != want.Content || got.Author != want.Author {
			t.Errorf(
				`did not get expected comment, got {Id="%s", Title="%s", Content="%s", Author="%s"} want {Id="%s", Title="%s", Content="%s", Author="%s"}`,
				got.Id,
				got.Post,
				got.Content,
				got.Author,
				want.Id,
				want.Post,
				want.Content,
				want.Author,
			)
		}
	})

	t.Run("returns 404", func(t *testing.T) {
		request := newStoreCommentRequest("3", `{"content": "Comment Content", "author": "Alex"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

		got := getErrorFromResponseModel(t, response.Body)
		want := meshtalk.ErrPostNotFound

		assertGotError(t, got, want)
	})

	t.Run("returns 400", func(t *testing.T) {

		t.Run("unsupported data", func(t *testing.T) {
			request := newStoreCommentRequest("1", `data`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := meshtalk.ErrUnsupportedComment

			assertGotError(t, got, want)
		})

		t.Run("missing fields error", func(t *testing.T) {
			request := newStoreCommentRequest("1", `{}`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := meshtalk.ErrMissingCommentFields

			assertGotError(t, got, want)
		})
	})
}

func TestPUTComments(t *testing.T) {
	storage := &StubStorage{
		posts: map[string]entities.Post{
			"1": {
				Id:        "1",
				Title:     "Post 1",
				Content:   "Post Content",
				Author:    "Alex",
				CreatedAt: newDate(2023, time.December, 4, 16, 30, 30, 100),
			},
		},
		comments: map[string]map[string]entities.Comment{
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
		},
	}
	server := meshtalk.NewServer(storage)

	t.Run("returns 204", func(t *testing.T) {
		request := newEditCommentRequest("1", "1", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if len(storage.commentEditCalls) != 1 {
			t.Error("didn't edit the comment")
		}
	})

	t.Run("returns 404", func(t *testing.T) {
		request := newEditCommentRequest("1", "3", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

		got := getErrorFromResponseModel(t, response.Body)
		want := meshtalk.ErrCommentNotFound

		assertGotError(t, got, want)
	})

	t.Run("returns 500", func(t *testing.T) {
		storage := &StubFailingStorage{}
		server := meshtalk.NewServer(storage)
		request := newEditCommentRequest("1", "2", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestServerTimeout(t *testing.T) {
	t.Run("returns 408 when reaches server timeout", func(t *testing.T) {
		storage := &MockStorage{
			GetPostFunc: func(id string) *entities.Post {
				time.Sleep(time.Second * 2)
				return &entities.Post{}
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

func newDate(year int, month time.Month, day, hour, min, sec, mlsec int) string {
	d := time.Date(year, month, day, hour, min, sec, mlsec*1e6, time.UTC)
	b, _ := d.MarshalText()
	return string(b)
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

func newGetCommentsRequest(postId, commentId string) *http.Request {
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

func newGetPostCommentsRequest(postId, commentId string) *http.Request {
	url := "/posts/" + postId + "/comments"
	if commentId != "" {
		url += "/" + commentId
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	return req
}

func newStoreCommentRequest(post, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/posts/"+post+"/comments", strings.NewReader(jsonRaw))
	return req
}

func newEditCommentRequest(postId, commentId, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPut, "/posts/"+postId+"/comments/"+commentId, strings.NewReader(jsonRaw))
	return req
}

func assertStatus(t testing.TB, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("did not get correct status, got %d but want %d", response.Code, want)
	}
}

func assertGotPost(t testing.TB, got, want entities.Post) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong post received, got %v but want %v", got, want)
	}
}

func assertGotError(t testing.TB, got meshtalk.Error, want *meshtalk.Error) {
	t.Helper()

	if !reflect.DeepEqual(got, *want) {
		t.Errorf("got error %q, but want %q", got, *want)
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

func getPostFromResponseModel(t *testing.T, body io.Reader) entities.Post {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)

	var post entities.Post
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

func getCommentsListFromResponseModel(t *testing.T, body io.Reader) []entities.Comment {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)
	data, _ := json.Marshal(responseModel.Data)

	var list []entities.Comment
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&list); err != nil {
		t.Fatalf("unable to parse data into comments list, %v", err)
	}

	return list
}

func getCommentFromResponseModel(t *testing.T, body io.Reader) entities.Comment {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)
	data, _ := json.Marshal(responseModel.Data)

	var c entities.Comment
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&c); err != nil {
		t.Fatalf("unable to parse data into comments list, %v", err)
	}

	return c
}
