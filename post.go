package post

import (
	"fmt"
	"net/http"
	"strings"
)

type PostStorage interface {
	GetPost(is string) string
}

type PostServer struct {
	storage PostStorage
}

func (p *PostServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	postID := strings.TrimPrefix(r.URL.Path, "/posts/")

	foundPost := p.storage.GetPost(postID)

	if foundPost == "" {
		w.WriteHeader(http.StatusNotFound)
	}

	fmt.Fprint(w, foundPost)
}
