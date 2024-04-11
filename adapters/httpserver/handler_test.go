package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"meshtalk/domain/entities"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func getSessionCookie(response *httptest.ResponseRecorder) (*http.Cookie, error) {
	setCookie, ok := response.Header()["Set-Cookie"]
	if !ok || len(setCookie) == 0 {
		return nil, errors.New("missing set-cookie")
	}
	cookie := map[string]string{}
	for _, pair := range strings.Split(setCookie[0], "; ") {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			cookie[kv[0]] = kv[1]
			continue
		}
		cookie[kv[0]] = "true"
	}
	maxAge, _ := strconv.Atoi(cookie["Max-Age"])
	httpOnly, _ := strconv.ParseBool(cookie["HttpOnly"])
	cookieName := SessionCookieName()
	sessionCookie := &http.Cookie{
		Name:     cookieName,
		Value:    cookie[cookieName],
		Path:     cookie["Path"],
		HttpOnly: httpOnly,
		MaxAge:   maxAge,
	}
	expires, hasExpires := cookie["Expires"]
	if hasExpires {
		sessionCookie.Expires, _ = time.Parse(time.RFC1123, expires)
	}
	return sessionCookie, nil
}

func login(t *testing.T, server *Server) *http.Cookie {
	request, _ := http.NewRequest(http.MethodPost, "/login", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	cookie, err := getSessionCookie(response)
	if err != nil {
		t.Fatalf("cannot login, %v", err)
	}

	return cookie
}

func TestGETPosts(t *testing.T) {
	storage := &stubStorage{
		posts: map[string]entities.Post{
			"1": {
				Id:        "1",
				Title:     "Post 1",
				Content:   "Post Content",
				Author:    "1",
				CreatedAt: newDate(2023, time.December, 4, 16, 30, 30, 100),
			},
			"2": {
				Id:        "2",
				Title:     "Post 2",
				Content:   "Post Content",
				Author:    "2",
				CreatedAt: newDate(2023, time.December, 4, 17, 0, 0, 0),
			},
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns post with id equal to 1", func(t *testing.T) {

		request := newGetPostRequest(sessionCookie, "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		got := getPostFromResponseModel(t, response.Body)
		want := storage.posts["1"]

		assertGotPost(t, got, want)
	})

	t.Run("returns post with id equal to 2", func(t *testing.T) {

		request := newGetPostRequest(sessionCookie, "2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		got := getPostFromResponseModel(t, response.Body)
		want := storage.posts["2"]

		assertGotPost(t, got, want)
	})

	t.Run("returns 404 on nonexistent post", func(t *testing.T) {
		request := newGetPostRequest(sessionCookie, "0")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})

	t.Run("returns all posts", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/posts", nil)
		request.AddCookie(sessionCookie)
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

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		server := NewServer(&stubFailingStorage{})
		request := newGetPostRequest(sessionCookie, "0")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestPOSTPosts(t *testing.T) {
	storage := &stubStorage{
		customers: map[string]entities.Customer{
			"1": {
				Id:   "1",
				Name: "Alex",
			},
		},
		posts: map[string]entities.Post{},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run(`returns 201 and post after create post`, func(t *testing.T) {
		request := newCreatePostRequest(sessionCookie, `{"title": "Post X", "content": "Post Content", "author": "1"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusCreated)

		got := getPostFromResponseModel(t, response.Body)

		want := entities.Post{
			Id:      "1",
			Title:   "Post X",
			Content: "Post Content",
			Author:  "1",
		}

		if _, ok := storage.posts["1"]; !ok {
			t.Fatal("didn't creates the post")
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
	t.Run("returns 400", func(t *testing.T) {
		t.Run("unsupported error", func(t *testing.T) {
			request := newCreatePostRequest(sessionCookie, `data`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := ErrUnsupportedPost

			assertGotError(t, got, want)
		})
		t.Run("missing fields error", func(t *testing.T) {
			request := newCreatePostRequest(sessionCookie, `{}`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := ErrMissingPostFields

			assertGotError(t, got, want)
		})
		t.Run("nonexistent author", func(t *testing.T) {
			request := newCreatePostRequest(sessionCookie, `{"title": "Post X", "content": "Post Content", "author": "2"}`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := ErrNonexistentAuthor

			assertGotError(t, got, want)
		})
	})
	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &stubFailingStorage{}
		server := NewServer(storage)

		request := newCreatePostRequest(sessionCookie, `{"title": "Post X", "content": "Post Content", "author": "Alex"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)

	})
}

func newPost(id, title, content, author string) *entities.Post {
	return &entities.Post{
		Id:      id,
		Title:   title,
		Content: content,
		Author:  author,
	}
}

func TestPUTPosts(t *testing.T) {
	storage := &stubStorage{
		posts: map[string]entities.Post{
			"1": *newPost("1", "Post 1", "Post Content", "1"),
			"2": *newPost("2", "Post 2", "Post Content", "2"),
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns 204 on post edited", func(t *testing.T) {
		request := newEditPostRequest(sessionCookie, "1", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if !slices.Contains(
			storage.postEditCalls,
			`Id="1", Content="Edited Content"`,
		) {
			t.Errorf("didn't update post")
		}
	})
	t.Run("returns 404 on nonexistent post", func(t *testing.T) {
		request := newEditPostRequest(sessionCookie, "3", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})
	t.Run("returns 400 and nothing to update error", func(t *testing.T) {
		request := newEditPostRequest(sessionCookie, "1", `{}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusBadRequest)

		got := getErrorFromResponseModel(t, response.Body)
		assertGotError(t, got, ErrNothingToUpdate)
	})
	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &stubFailingStorage{
			posts: map[string]entities.Post{
				"1": *newPost("1", "Post 1", "Post Content", "1"),
			},
		}
		server := NewServer(storage)
		jsonRaw := `{"Content": "Edited Content"}`
		request := newEditPostRequest(sessionCookie, "1", jsonRaw)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestDELETEPosts(t *testing.T) {
	t.Run("returns 200 on post deleted", func(t *testing.T) {
		storage := &stubStorage{
			posts: map[string]entities.Post{
				"1": *newPost("1", "Post 1", "Post Content", "1"),
			},
		}
		server := NewServer(storage)

		sessionCookie := login(t, server)

		request := newDeletePostRequest(sessionCookie, "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)
		if len(storage.posts) != 0 {
			t.Errorf("expected that the post was deleted, but it was not")
		}
	})
	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &stubFailingStorage{}
		server := NewServer(storage)

		sessionCookie := login(t, server)

		request := newDeletePostRequest(sessionCookie, "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestGETComments(t *testing.T) {
	storage := &stubStorage{
		posts: map[string]entities.Post{
			"1": {
				Id:        "1",
				Title:     "Post 1",
				Content:   "Post Content",
				Author:    "1",
				CreatedAt: newDate(2023, time.December, 4, 16, 30, 30, 100),
			},
			"2": {
				Id:        "2",
				Title:     "Post 2",
				Content:   "Post Content",
				Author:    "2",
				CreatedAt: newDate(2023, time.December, 4, 17, 0, 0, 0),
			},
		},
		comments: map[string]map[string]entities.Comment{
			"1": {
				"1": {
					Id:        "1",
					Post:      "1",
					Content:   "Some comment",
					Author:    "3",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
				"2": {
					Id:        "2",
					Post:      "1",
					Content:   "Some comment",
					Author:    "4",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
			},
			"2": {
				"1": {
					Id:        "1",
					Post:      "2",
					Content:   "Some comment",
					Author:    "5",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
			},
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns comments from post 1", func(t *testing.T) {

		t.Run("for /comments?post=1", func(t *testing.T) {
			request := newGetCommentsRequest(sessionCookie, "1", "")
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
			request := newGetPostCommentsRequest(sessionCookie, "1", "")
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

		t.Run("for /comments?post=1&comment=2", func(t *testing.T) {
			request := newGetCommentsRequest(sessionCookie, "1", "2")
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
			request := newGetPostCommentsRequest(sessionCookie, "1", "2")
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusOK)

			got := getCommentFromResponseModel(t, response.Body)
			want := storage.comments["1"]["2"]

			assertGotComment(t, got, want)
			// if !reflect.DeepEqual(got, want) {
			// 	t.Errorf("got comment %v, but want %v", got, want)
			// }
		})
	})

	t.Run("returns 404 when try to get comments from post 3", func(t *testing.T) {
		request := newGetPostCommentsRequest(sessionCookie, "3", "")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})

	t.Run("returns 404 when try to get comment 3 from post 2", func(t *testing.T) {
		request := newGetPostCommentsRequest(sessionCookie, "2", "3")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})

	t.Run("returns all comments", func(t *testing.T) {
		request := newGetCommentsRequest(sessionCookie, "", "")
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

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		server := NewServer(&stubFailingStorage{})
		request := newGetCommentsRequest(sessionCookie, "1", "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestPOSTComments(t *testing.T) {
	storage := &stubStorage{
		customers: map[string]entities.Customer{
			"1": {
				Id:   "1",
				Name: "Alex",
			},
		},
		posts: map[string]entities.Post{
			"1": {
				Id:        "1",
				Title:     "Post 1",
				Content:   "Post Content",
				Author:    "1",
				CreatedAt: newDate(2023, time.December, 4, 16, 30, 30, 100),
			},
			"2": {
				Id:        "2",
				Title:     "Post 2",
				Content:   "Post Content",
				Author:    "2",
				CreatedAt: newDate(2023, time.December, 4, 17, 0, 0, 0),
			},
		},
		comments: map[string]map[string]entities.Comment{},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run(`returns 201 and comment after create comment`, func(t *testing.T) {
		request := newCreateCommentRequest(sessionCookie, "1", `{"content": "Comment Content", "author": "1"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusCreated)

		got := getCommentFromResponseModel(t, response.Body)
		want := entities.Comment{
			Id:      "1",
			Post:    "1",
			Content: "Comment Content",
			Author:  "1",
		}

		comments, ok := storage.comments["1"]
		if !ok {
			t.Fatal("didn't contains any comment into post 1")
		}

		if _, ok := comments["1"]; !ok {
			t.Fatal("didn't creates the comment")
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

	t.Run("returns 404 because the post doesn't exist", func(t *testing.T) {
		request := newCreateCommentRequest(sessionCookie, "3", `{"content": "Comment Content", "author": "1"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})

	t.Run("returns 400", func(t *testing.T) {

		t.Run("unsupported data", func(t *testing.T) {
			request := newCreateCommentRequest(sessionCookie, "1", `data`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := ErrUnsupportedComment

			assertGotError(t, got, want)
		})

		t.Run("missing fields error", func(t *testing.T) {
			request := newCreateCommentRequest(sessionCookie, "1", `{}`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := ErrMissingCommentFields

			assertGotError(t, got, want)
		})

		t.Run("nonexistent author", func(t *testing.T) {
			request := newCreateCommentRequest(sessionCookie, "1", `{"content": "Comment Content", "author": "2"}`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := ErrNonexistentAuthor

			assertGotError(t, got, want)
		})
	})

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		server := NewServer(&stubFailingStorage{})
		request := newCreateCommentRequest(sessionCookie, "1", `{"content": "Comment Content", "author": "1"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestPUTComments(t *testing.T) {
	storage := &stubStorage{
		posts: map[string]entities.Post{
			"1": {
				Id:        "1",
				Title:     "Post 1",
				Content:   "Post Content",
				Author:    "1",
				CreatedAt: newDate(2023, time.December, 4, 16, 30, 30, 100),
			},
		},
		comments: map[string]map[string]entities.Comment{
			"1": {
				"1": {
					Id:        "1",
					Post:      "1",
					Content:   "Some comment",
					Author:    "2",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
				"2": {
					Id:        "2",
					Post:      "1",
					Content:   "Some comment",
					Author:    "3",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
			},
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns 204", func(t *testing.T) {
		request := newEditCommentRequest(sessionCookie, "1", "1", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if !slices.Contains(
			storage.commentEditCalls,
			`Post="1", Id="1", Content="Edited Content"`,
		) {
			t.Errorf("didn't update comment")
		}
	})
	t.Run("returns 404 on nonexistent comment", func(t *testing.T) {
		request := newEditCommentRequest(sessionCookie, "1", "3", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})
	t.Run("returns 404 because the post doesn't exist", func(t *testing.T) {
		request := newEditCommentRequest(sessionCookie, "2", "1", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})
	t.Run("returns 400 and nothing to update error", func(t *testing.T) {
		request := newEditCommentRequest(sessionCookie, "1", "1", `{}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusBadRequest)

		got := getErrorFromResponseModel(t, response.Body)
		assertGotError(t, got, ErrNothingToUpdate)
	})
	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &stubFailingStorage{}
		server := NewServer(storage)
		request := newEditCommentRequest(sessionCookie, "1", "2", `{"Content": "Edited Content"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestDELETEComments(t *testing.T) {
	storage := &stubStorage{
		posts: map[string]entities.Post{
			"1": {
				Id:        "1",
				Title:     "Post 1",
				Content:   "Post Content",
				Author:    "1",
				CreatedAt: newDate(2023, time.December, 4, 16, 30, 30, 100),
			},
		},
		comments: map[string]map[string]entities.Comment{
			"1": {
				"1": {
					Id:        "1",
					Post:      "1",
					Content:   "Some comment",
					Author:    "2",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
				"2": {
					Id:        "2",
					Post:      "1",
					Content:   "Some comment",
					Author:    "3",
					CreatedAt: newDate(2024, time.January, 23, 12, 30, 30, 100),
				},
			},
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns 204", func(t *testing.T) {
		request := newDeleteCommentRequest(sessionCookie, "1", "2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if _, ok := storage.comments["1"]["2"]; ok {
			t.Errorf("expected that the comment was deleted, but it was not")
		}
	})

	t.Run("returns 404 because the post doesn't exist", func(t *testing.T) {
		request := newDeleteCommentRequest(sessionCookie, "2", "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &stubFailingStorage{}
		server := NewServer(storage)
		request := newDeleteCommentRequest(sessionCookie, "1", "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestPOSTCustomers(t *testing.T) {
	storage := NewStubStorage()
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns 201 and customer", func(t *testing.T) {
		request := newCreateCustomerRequest(sessionCookie, `{"tag": "marie", "name": "Marie", "password": "123456"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusCreated)

		if _, ok := storage.customers["1"]; !ok {
			t.Errorf("didn't creates the customer")
		}

		want := entities.Customer{
			Id:   "1",
			Tag:  "marie",
			Name: "Marie",
		}

		got := getCustomerFromResponseModel(t, response.Body)

		if got.Id != want.Id || got.Tag != want.Tag || got.Name != want.Name {
			t.Errorf(
				`did not get expected comment, got {Id="%s", Tag="%s", Name="%s"} want {Id="%s", Tag="%s", Name="%s"}`,
				got.Id,
				got.Tag,
				got.Name,
				want.Id,
				want.Tag,
				want.Name,
			)
		}
	})

	t.Run("returns 400", func(t *testing.T) {
		t.Run("unsupported error", func(t *testing.T) {
			request := newCreateCustomerRequest(sessionCookie, `data`)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertStatus(t, response, http.StatusBadRequest)

			got := getErrorFromResponseModel(t, response.Body)
			want := ErrUnsupportedCustomer

			assertGotError(t, got, want)
		})

		t.Run("missing fields error", func(t *testing.T) {

			cases := []string{
				`{}`,
				`{"tag": "james"}`,
				`{"name": "James"}`,
				`{"password": "123456"}`,
				`{"tag": "james", "name": "James"}`,
				`{"name": "James", "passaword": "123456"}`,
				`{"tag": "james", "password": "123456"}`,
			}

			for _, raw := range cases {
				request := newCreateCustomerRequest(sessionCookie, raw)
				response := httptest.NewRecorder()

				server.ServeHTTP(response, request)

				assertStatus(t, response, http.StatusBadRequest)

				got := getErrorFromResponseModel(t, response.Body)
				want := ErrMissingCustomerFields

				assertGotError(t, got, want)
			}
		})
	})

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		server := NewServer(&stubFailingStorage{})

		request := newCreateCustomerRequest(sessionCookie, `{"name": "Marie"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestGETCustomers(t *testing.T) {
	storage := &stubStorage{
		customers: map[string]entities.Customer{
			"1": {
				Id:        "1",
				Name:      "John",
				CreatedAt: newDate(2024, time.April, 4, 11, 55, 0, 0),
			},
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns 200 and the customer data", func(t *testing.T) {
		request := newGetCustomerRequest(sessionCookie, "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusOK)

		got := getCustomerFromResponseModel(t, response.Body)
		want := storage.customers["1"]

		assertGotCustomer(t, got, want)
	})

	t.Run("returns 404 on nonexistent customer", func(t *testing.T) {
		request := newGetCustomerRequest(sessionCookie, "2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		server := NewServer(&stubFailingStorage{})

		request := newGetCustomerRequest(sessionCookie, "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestPUTCustomers(t *testing.T) {
	storage := &stubStorage{
		customers: map[string]entities.Customer{
			"1": {
				Id:        "1",
				Name:      "John",
				CreatedAt: newDate(2024, time.April, 4, 11, 55, 0, 0),
			},
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns 204", func(t *testing.T) {
		request := newEditCustomerRequest(sessionCookie, "1", `{"name": "Jhonny"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if !slices.Contains(
			storage.customerEditCalls,
			`Id="1", Name="Jhonny"`,
		) {
			t.Errorf("didn't update customer")
		}
	})

	t.Run("returns 404 on nonexistent customer", func(t *testing.T) {
		request := newEditCustomerRequest(sessionCookie, "2", `{"name": "Marie"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNotFound)

	})

	t.Run("returns 400 and nothing to update error", func(t *testing.T) {
		request := newEditCustomerRequest(sessionCookie, "1", `{}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusBadRequest)

		got := getErrorFromResponseModel(t, response.Body)
		assertGotError(t, got, ErrNothingToUpdate)
	})

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		server := NewServer(&stubFailingStorage{})

		request := newEditCustomerRequest(sessionCookie, "1", `{"name": "James"}`)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestDELETECustomers(t *testing.T) {
	storage := &stubStorage{
		customers: map[string]entities.Customer{
			"1": {
				Id:        "1",
				Name:      "John",
				CreatedAt: newDate(2024, time.April, 4, 11, 55, 0, 0),
			},
			"2": {
				Id:        "2",
				Name:      "Mary",
				CreatedAt: newDate(2024, time.April, 4, 11, 55, 0, 0),
			},
		},
	}
	server := NewServer(storage)

	sessionCookie := login(t, server)

	t.Run("returns 204", func(t *testing.T) {
		request := newDeleteCustomerRequest(sessionCookie, "2")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusNoContent)

		if _, ok := storage.customers["2"]; ok {
			t.Errorf("didn't delete customer")
		}
	})

	t.Run("returns 500 on unexpected error", func(t *testing.T) {
		storage := &stubFailingStorage{}
		server := NewServer(storage)

		request := newDeleteCustomerRequest(sessionCookie, "1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response, http.StatusInternalServerError)
	})
}

func TestServerTimeout(t *testing.T) {
	t.Run("returns 408 when reaches server timeout", func(t *testing.T) {
		storage := &mockStorage{
			GetPostFunc: func(id string) (*entities.Post, error) {
				time.Sleep(time.Second * 2)
				return &entities.Post{}, nil
			},
		}
		server := NewServer(storage)
		server.SetTimeout(time.Second * 1)

		sessionCookie := login(t, server)

		request := newGetPostRequest(sessionCookie, "1")
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

func newGetPostRequest(cookie *http.Cookie, id string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/posts/"+id, nil)
	req.AddCookie(cookie)
	return req
}

func newCreatePostRequest(cookie *http.Cookie, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/posts", strings.NewReader(jsonRaw))
	req.AddCookie(cookie)
	return req
}

func newEditPostRequest(cookie *http.Cookie, id, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPut, "/posts/"+id, strings.NewReader(jsonRaw))
	req.AddCookie(cookie)
	return req
}

func newDeletePostRequest(cookie *http.Cookie, id string) *http.Request {
	req, _ := http.NewRequest(http.MethodDelete, "/posts/"+id, nil)
	req.AddCookie(cookie)
	return req
}

func newGetCommentsRequest(cookie *http.Cookie, postId, commentId string) *http.Request {
	url := "/comments"
	if postId != "" {
		url = url + "?post=" + postId
		if commentId != "" {
			url = url + "&comment=" + commentId
		}
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(cookie)
	return req
}

func newGetPostCommentsRequest(cookie *http.Cookie, postId, commentId string) *http.Request {
	url := "/posts/" + postId + "/comments"
	if commentId != "" {
		url += "/" + commentId
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(cookie)
	return req
}

func newCreateCommentRequest(cookie *http.Cookie, post, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/posts/"+post+"/comments", strings.NewReader(jsonRaw))
	req.AddCookie(cookie)
	return req
}

func newEditCommentRequest(cookie *http.Cookie, postId, commentId, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPut, "/posts/"+postId+"/comments/"+commentId, strings.NewReader(jsonRaw))
	req.AddCookie(cookie)
	return req
}

func newDeleteCommentRequest(cookie *http.Cookie, postId, commentId string) *http.Request {
	req, _ := http.NewRequest(http.MethodDelete, "/posts/"+postId+"/comments/"+commentId, nil)
	req.AddCookie(cookie)
	return req
}

func newCreateCustomerRequest(cookie *http.Cookie, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/customers", strings.NewReader(jsonRaw))
	req.AddCookie(cookie)
	return req
}

func newGetCustomerRequest(cookie *http.Cookie, customerId string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/customers/"+customerId, nil)
	req.AddCookie(cookie)
	return req
}

func newEditCustomerRequest(cookie *http.Cookie, customerId, jsonRaw string) *http.Request {
	req, _ := http.NewRequest(http.MethodPut, "/customers/"+customerId, strings.NewReader(jsonRaw))
	req.AddCookie(cookie)
	return req
}

func newDeleteCustomerRequest(cookie *http.Cookie, customerId string) *http.Request {
	req, _ := http.NewRequest(http.MethodDelete, "/customers/"+customerId, nil)
	req.AddCookie(cookie)
	return req
}

func getResponseModelFromResponse(t *testing.T, body io.Reader) ResponseModel {
	t.Helper()

	var responseModel ResponseModel
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

func getErrorFromResponseModel(t *testing.T, body io.Reader) Error {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)

	var responseError Error
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
		t.Fatalf("unable to parse data from ResponseModel into Comment, %v", err)
	}

	return c
}

func getCustomerFromResponseModel(t *testing.T, body io.Reader) entities.Customer {
	t.Helper()

	responseModel := getResponseModelFromResponse(t, body)
	data, _ := json.Marshal(responseModel.Data)

	var c entities.Customer
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&c); err != nil {
		t.Fatalf("unable to parse data from ResponseModel into Customer, %v", err)
	}

	return c
}
