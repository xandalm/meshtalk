package meshtalk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Storage interface {
	GetPost(id string) *Post
	StorePost(post *Post) string
	EditPost(post *Post) bool
}

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
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}

}

func (s *Server) storePostHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := s.storage.StorePost(&post)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, id)
}

func (s *Server) getPostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)

	foundPost := s.storage.GetPost(postID)

	if foundPost == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(foundPost)
}

func (s *Server) editPostHandler(w http.ResponseWriter, r *http.Request) {
	postID := s.extractPostIdFromURLPath(r)

	var post Post
	json.NewDecoder(r.Body).Decode(&post)
	post.Id = postID

	if s.storage.EditPost(&post) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) extractPostIdFromURLPath(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Path, "/posts/")
}
