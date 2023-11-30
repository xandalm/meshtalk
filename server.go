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
	postID := strings.TrimPrefix(r.URL.Path, "/posts/")

	foundPost := s.storage.GetPost(postID)

	if foundPost == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(foundPost)
}
