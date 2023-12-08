package meshtalk

import (
	"encoding/json"
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

type ResponseModel struct {
	Data  any `json:"data,omitempty"`
	Error any `json:"error,omitempty"`
}

var (
	ErrPostNotFound         = Error{"there is no such post here"}
	ErrNotSupportedPostData = Error{"unsupported data to parse into post"}
)

type Server struct {
	storage Storage
}

func NewServer(storage Storage) *Server {
	return &Server{storage}
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
		s.writeResponseModel(w, nil, ErrNotSupportedPostData)
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

func (s *Server) writeResponseModel(w http.ResponseWriter, data any, err any) {
	toJSON(
		w,
		ResponseModel{
			Data:  data,
			Error: err,
		},
	)
}

func (s *Server) getPostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)

	foundPost := s.storage.GetPost(postID)

	if foundPost == nil {
		w.WriteHeader(http.StatusNotFound)
		s.writeResponseModel(w, nil, ErrPostNotFound)
		return
	}
	s.writeResponseModel(w, *foundPost, nil)
}

func (s *Server) editPostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)

	var post Post
	fromJSON(r.Body, &post)
	post.Id = postID

	foundPost := s.storage.GetPost(postID)
	if foundPost == nil {
		w.WriteHeader(http.StatusNotFound)
		s.writeResponseModel(w, nil, ErrPostNotFound)
		return
	}

	if err := s.storage.EditPost(&post); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
