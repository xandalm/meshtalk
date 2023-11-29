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

	fmt.Fprint(w, p.storage.GetPost(postID))
}

func GetPost(id string) string {
	if id == "1" {
		return `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`
	}
	if id == "2" {
		return `{"ID": "2", "Title": "Post 2", "Content": "Post Content"}`
	}
	return ""
}
