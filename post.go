package post

import (
	"fmt"
	"net/http"
)

func GetPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`)
}
