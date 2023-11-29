package post

import (
	"fmt"
	"net/http"
	"strings"
)

func GetPost(w http.ResponseWriter, r *http.Request) {
	postID := strings.TrimPrefix(r.URL.Path, "/posts/")
	if postID == "1" {
		fmt.Fprint(w, `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`)
		return
	}
	if postID == "2" {
		fmt.Fprint(w, `{"ID": "2", "Title": "Post 2", "Content": "Post Content"}`)
		return
	}
}
