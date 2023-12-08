package meshtalk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Storage interface {
	GetPost(id string) *Post
	StorePost(post *Post) string
	EditPost(post *Post) bool
	DeletePost(id string) bool
}

type Error struct {
	Message string `json:"message,omitempty"`
}

type ResponseModel struct {
	Data  any `json:"data,omitempty"`
	Error any `json:"error,omitempty"`
}

var (
	ErrPostNotFound = Error{"there is no such post here"}
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
		return
	}
	id := s.storage.StorePost(&post)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, id)
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

	if s.storage.EditPost(&post) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func (s *Server) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)
	if s.storage.DeletePost(postID) {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
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
