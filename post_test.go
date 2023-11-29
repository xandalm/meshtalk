package post

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPost(t *testing.T) {

	request, _ := http.NewRequest(http.MethodGet, "/post/1", nil)
	response := httptest.NewRecorder()

	GetPost(response, request)

	got := response.Body.String()
	want := `{"ID": "1", "Title": "Post 1", "Content": "Post Content"}`

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
