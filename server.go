package meshtalk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	router "github.com/xandalm/go-router"
)

type Storage interface {
	GetPost(id string) *Post
	GetPosts() []Post
	StorePost(post *Post) error
	EditPost(post *Post) error
	DeletePost(id string) error
	GetComments(post string) []Comment
	GetComment(post, id string) *Comment
	StoreComment(comment *Comment) error
}

type Error struct {
	Message string `json:"message,omitempty"`
}

func NewError(message string) *Error {
	return &Error{
		Message: message,
	}
}

type ResponseModel struct {
	Data  any `json:"data,omitempty"`
	Error any `json:"error,omitempty"`
}

const (
	ErrPostNotFoundMessage         = "there is no such post here"
	ErrUnsupportedPostMessage      = "unsupported data to parse as post"
	ErrMissingPostFieldsMessage    = "missing post fields (title, content and author are required)"
	ErrUnsupportedCommentMessage   = "unsupported data to parse as comment"
	ErrMissingCommentFieldsMessage = "missing comment fields (content and author are required)"
)

var (
	ErrPostNotFound         = errors.New("ERR_POST_NOT_FOUND")
	ErrUnsupportedPost      = errors.New("ERR_UNSUPPORTED_POST")
	ErrMissingPostFields    = errors.New("ERR_MISSING_POST_FIELDS")
	ErrUnsupportedComment   = errors.New("ERR_UNSUPPORTED_COMMENT")
	ErrMissingCommentFields = errors.New("ERR_MISSING_COMMENT_FIELDS")
)

type Server struct {
	storage Storage
	router  *router.Router
	to      time.Duration
}

func NewServer(storage Storage) *Server {
	s := &Server{
		storage: storage,
		router:  &router.Router{},
		to:      time.Minute,
	}

	s.router.GetFunc("/posts/{id}", s.getPostHandler)
	s.router.PutFunc("/posts/{id}", s.editPostHandler)
	s.router.DeleteFunc("/posts/{id}", s.deletePostHandler)
	s.router.GetFunc("/posts", s.getPostHandler)
	s.router.PostFunc("/posts", s.storePostHandler)

	s.router.GetFunc("/posts/{pid}/comments/{cid}", s.getPostCommentsHandler)
	s.router.GetFunc("/posts/{pid}/comments", s.getPostCommentsHandler)
	s.router.PostFunc("/posts/{pid}/comments", s.storePostCommentHandler)

	s.router.GetFunc("/comments", s.getCommentsHandler)

	return s
}

func (s *Server) SetTimeout(duration time.Duration) error {
	if duration < time.Second {
		return errors.New("timeout duration must be greater than 1s")
	}
	s.to = duration
	return nil
}

func (s *Server) writeResponseModel(w http.ResponseWriter, data any, err any) {
	toJSON(
		w,
		ResponseModel{
			Data:  data,
			Error: err,
		},
	)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())

	time.AfterFunc(s.to, cancel)
	r = r.WithContext(ctx)

	c := make(chan struct{})
	go func() {
		s.router.ServeHTTP(w, r)
		close(c)
	}()

	select {
	case <-ctx.Done():
		w.WriteHeader(http.StatusRequestTimeout)
	case <-c:
		return
	}
}

func (s *Server) storePostHandler(w router.ResponseWriter, r *router.Request) {
	var post Post
	err := r.ParseBodyInto(&post)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.writeResponseModel(w, nil, NewError(ErrUnsupportedPostMessage))
		return
	}

	if post.Title == "" || post.Content == "" || post.Author == "" {
		w.WriteHeader(http.StatusBadRequest)
		s.writeResponseModel(w, nil, NewError(ErrMissingPostFieldsMessage))
		return
	}

	if err := s.storage.StorePost(&post); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeResponseModel(
		w,
		post,
		nil,
	)
}

func (s *Server) getPostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]

	if postId == "" {
		w.WriteHeader(http.StatusOK)
		s.writeResponseModel(w, s.storage.GetPosts(), nil)
		return
	}

	foundPost := s.storage.GetPost(postId)

	if foundPost == nil {
		w.WriteHeader(http.StatusNotFound)
		s.writeResponseModel(w, nil, NewError(ErrPostNotFoundMessage))
		return
	}
	s.writeResponseModel(w, *foundPost, nil)
}

func (s *Server) editPostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]

	var post Post
	err := r.ParseBodyInto(&post)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.writeResponseModel(w, nil, NewError(ErrUnsupportedPostMessage))
		return
	}
	post.Id = postId

	if err := s.storage.EditPost(&post); err != nil {

		if err == ErrPostNotFound {
			w.WriteHeader(http.StatusNotFound)
			s.writeResponseModel(w, nil, NewError(ErrPostNotFoundMessage))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deletePostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]
	if err := s.storage.DeletePost(postId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getCommentsHandler(w router.ResponseWriter, r *router.Request) {
	var comments []Comment

	query := r.URL.Query()

	post, hasPost := query["post"]

	if hasPost {
		id, hasId := query["id"]
		if hasId {
			found := s.storage.GetComment(post[0], id[0])
			if found != nil {
				comments = append(comments, *found)
			}
		} else {
			comments = s.storage.GetComments(post[0])
		}
	} else {
		comments = s.storage.GetComments("")
	}
	w.WriteHeader(http.StatusOK)
	s.writeResponseModel(w, comments, nil)
}

func (s *Server) getPostCommentsHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	pid := params["pid"]
	cid := params["cid"]

	post := s.storage.GetPost(pid)

	if post == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if cid != "" {
		comment := s.storage.GetComment(pid, cid)
		if comment == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s.writeResponseModel(w, comment, nil)
		return
	}
	s.writeResponseModel(w, s.storage.GetComments(pid), nil)
}

func (s *Server) storePostCommentHandler(w router.ResponseWriter, r *router.Request) {
	var comment Comment

	err := r.ParseBodyInto(&comment)

	pid := r.Params()["pid"]

	post := s.storage.GetPost(pid)

	if post == nil {
		w.WriteHeader(http.StatusNotFound)
		s.writeResponseModel(w, nil, NewError(ErrPostNotFoundMessage))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.writeResponseModel(w, nil, NewError(ErrUnsupportedCommentMessage))
		return
	}

	if comment.Content == "" || comment.Author == "" {
		w.WriteHeader(http.StatusBadRequest)
		s.writeResponseModel(w, nil, NewError(ErrMissingCommentFieldsMessage))
		return
	}

	comment.Post = post.Id

	if err := s.storage.StoreComment(&comment); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeResponseModel(w, comment, nil)
}

func toJSON(w io.Writer, s any) error {
	return json.NewEncoder(w).Encode(s)
}
