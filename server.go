package meshtalk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	EditComment(comment *Comment) error
}

type Error struct {
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewError(name, message string) *Error {
	return &Error{
		Name:    name,
		Message: message,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf(`[%s "%s"]`, e.Name, e.Message)
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
	ErrCommentNotFoundMessage      = "there is no such comment here"
)

var (
	ErrPostNotFound         = NewError("ERR_POST_NOT_FOUND", ErrPostNotFoundMessage)
	ErrUnsupportedPost      = NewError("ERR_UNSUPPORTED_POST", ErrUnsupportedPostMessage)
	ErrMissingPostFields    = NewError("ERR_MISSING_POST_FIELDS", ErrMissingPostFieldsMessage)
	ErrCommentNotFound      = NewError("ERR_COMMENT_NOT_FOUND", ErrCommentNotFoundMessage)
	ErrUnsupportedComment   = NewError("ERR_UNSUPPORTED_COMMENT", ErrUnsupportedCommentMessage)
	ErrMissingCommentFields = NewError("ERR_MISSING_COMMENT_FIELDS", ErrMissingCommentFieldsMessage)
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
	s.router.PutFunc("/posts/{pid}/comments/{cid}", s.editPostCommentsHandler)
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

func (s *Server) writeResponse(w http.ResponseWriter, data, err any) {
	if err != nil {
		switch err {
		case ErrPostNotFound,
			ErrCommentNotFound:
			w.WriteHeader(http.StatusNotFound)
		case ErrMissingPostFields,
			ErrMissingCommentFields,
			ErrUnsupportedPost,
			ErrUnsupportedComment:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	writeJSON(
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
		s.writeResponse(w, nil, ErrUnsupportedPost)
		return
	}

	if post.Title == "" || post.Content == "" || post.Author == "" {
		s.writeResponse(w, nil, ErrMissingPostFields)
		return
	}

	if err := s.storage.StorePost(&post); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeResponse(
		w,
		post,
		nil,
	)
}

func (s *Server) getPostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]

	if postId == "" {
		s.writeResponse(w, s.storage.GetPosts(), nil)
		return
	}

	foundPost := s.storage.GetPost(postId)

	if foundPost == nil {
		s.writeResponse(w, nil, ErrPostNotFound)
		return
	}
	s.writeResponse(w, *foundPost, nil)
}

func (s *Server) editPostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]

	var post Post
	err := r.ParseBodyInto(&post)
	if err != nil {
		s.writeResponse(w, nil, ErrUnsupportedPost)
		return
	}
	post.Id = postId

	if err := s.storage.EditPost(&post); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deletePostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]
	if err := s.storage.DeletePost(postId); err != nil {
		s.writeResponse(w, nil, err)
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
	s.writeResponse(w, comments, nil)
}

func (s *Server) getPostCommentsHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	pid := params["pid"]
	cid := params["cid"]

	post := s.storage.GetPost(pid)

	if post == nil {
		s.writeResponse(w, nil, ErrPostNotFound)
		return
	}

	if cid != "" {
		comment := s.storage.GetComment(pid, cid)
		if comment == nil {
			s.writeResponse(w, nil, ErrCommentNotFound)
			return
		}
		s.writeResponse(w, comment, nil)
		return
	}
	s.writeResponse(w, s.storage.GetComments(pid), nil)
}

func (s *Server) storePostCommentHandler(w router.ResponseWriter, r *router.Request) {
	pid := r.Params()["pid"]

	post := s.storage.GetPost(pid)

	if post == nil {
		s.writeResponse(w, nil, ErrPostNotFound)
		return
	}

	var comment Comment
	err := r.ParseBodyInto(&comment)

	if err != nil {
		s.writeResponse(w, nil, ErrUnsupportedComment)
		return
	}

	if comment.Content == "" || comment.Author == "" {
		s.writeResponse(w, nil, ErrMissingCommentFields)
		return
	}

	comment.Post = post.Id

	if err := s.storage.StoreComment(&comment); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeResponse(w, comment, nil)
}

func (s *Server) editPostCommentsHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	var comment Comment
	if err := r.ParseBodyInto(&comment); err != nil {
		s.writeResponse(w, nil, ErrUnsupportedComment)
		return
	}

	comment.Post = params["pid"]
	comment.Id = params["cid"]

	if err := s.storage.EditComment(&comment); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w io.Writer, s any) error {
	return json.NewEncoder(w).Encode(s)
}
