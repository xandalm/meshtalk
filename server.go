package meshtalk

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type Storage interface {
	GetPost(id string) *Post
	StorePost(post *Post) error
	EditPost(post *Post) error
	DeletePost(id string) error
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
	ErrPostNotFoundMessage      = "there is no such post here"
	ErrUnsupportedPostMessage   = "unsupported data to parse into post"
	ErrMissingPostFieldsMessage = "missing post fields (title, content and author are required)"
)

var (
	ErrPostNotFound      = errors.New("ERR_POST_NOT_FOUND")
	ErrUnsupportedPost   = errors.New("ERR_UNSUPPORTED_POST")
	ErrMissingPostFields = errors.New("ERR_MISSING_POST_FIELDS")
)

type Server struct {
	storage Storage
}

func NewServer(storage Storage) *Server {
	return &Server{storage}
}

func (s *Server) writeResponseModel(w http.ResponseWriter, data any, err *Error) {
	toJSON(
		w,
		ResponseModel{
			Data:  data,
			Error: err,
		},
	)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		s.storePostHandler(w, r)
	case http.MethodGet:
		s.getPostHandler(w, r)
	case http.MethodPut:
		s.editPostHandler(w, r)
	case http.MethodDelete:
		s.deletePostHandler(w, r)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}

}

func (s *Server) storePostHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	err := fromJSON(r.Body, &post)
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

func (s *Server) getPostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)

	foundPost := s.storage.GetPost(postID)

	if foundPost == nil {
		w.WriteHeader(http.StatusNotFound)
		s.writeResponseModel(w, nil, NewError(ErrPostNotFoundMessage))
		return
	}
	s.writeResponseModel(w, *foundPost, nil)
}

func (s *Server) editPostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)

	var post Post
	fromJSON(r.Body, &post)
	post.Id = postID

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

func (s *Server) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)
	if err := s.storage.DeletePost(postID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) extractPostIdFromURLPath(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Path, "/posts/")
}

func toJSON(w io.Writer, s any) error {
	return json.NewEncoder(w).Encode(s)
}

func fromJSON(r io.Reader, s any) error {
	return json.NewDecoder(r).Decode(s)
}
