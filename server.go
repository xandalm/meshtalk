package meshtalk

import (
	"fmt"
	"net/http"
	"strings"
)

type Storage interface {
	GetPost(is string) string
}

type Server struct {
	storage Storage
}

func (p *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	postID := strings.TrimPrefix(r.URL.Path, "/posts/")

	foundPost := p.storage.GetPost(postID)

	if foundPost == "" {
		w.WriteHeader(http.StatusNotFound)
	}

	fmt.Fprint(w, foundPost)
}
