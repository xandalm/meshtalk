package post

import (
	"fmt"
	"net/http"
	"strings"
)

func PostServer(w http.ResponseWriter, r *http.Request) {
	postID := strings.TrimPrefix(r.URL.Path, "/posts/")

	fmt.Fprint(w, GetPost(postID))
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
