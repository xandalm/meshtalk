package meshtalk

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Storage interface {
	GetPost(id string) string
	StorePost(post string) string
}

type Server struct {
	storage Storage
}

func NewServer(storage Storage) *Server {
	return &Server{storage}
}

func (p *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {
		post, _ := io.ReadAll(r.Body)
		id := p.storage.StorePost(string(post))
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, id)
		return
	}

	postID := strings.TrimPrefix(r.URL.Path, "/posts/")

	foundPost := p.storage.GetPost(postID)

	if foundPost == "" {
		w.WriteHeader(http.StatusNotFound)
	}

	fmt.Fprint(w, foundPost)
}
