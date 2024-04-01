package specifications

import (
	"encoding/json"
	"fmt"
	"meshtalk/adapters/httpserver"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type CreatePostAction interface {
	CreatePost(args ...string) (int, string, error)
}

type ReadingPostAction interface {
	ReadPost(id string) (int, string, error)
}

var keepingPostId string

func SuccessfullyCreatePost(t testing.TB, driver CreatePostAction) {
	status, got, err := driver.CreatePost(`title: "Test Post"`, `content: "Some content"`, `author: "Someone"`)
	if err != nil {
		t.Errorf("failed specification test, %v", err)
	}
	assertHTTPStatus(t, status, http.StatusCreated)
	want := map[string]any{
		"title":   "Test Post",
		"content": "Some content",
		"author":  "Someone",
	}
	v := decodeByJSON(t, got)
	json, _ := v.(map[string]any)
	assertJSONHasNoError(t, json)
	assertJSONHasData(t, json)
	data, ok := json["data"].(map[string]any)
	if !ok {
		t.Fatalf("didn't get expected data type")
	}
	assertPostsCanBeTheSame(t, data, want)
	id, ok := data["id"]
	if !ok {
		t.Error("doesn't contain id in data")
	}
	keepingPostId = id.(string)
}

func UnableToCreatePostDueToMissingRequiredValues(t testing.TB, driver CreatePostAction) {
	cases := [][]string{
		{},
		{`title: "Test Post"`},
		{`content: "Some content"`},
		{`author: "Someone"`},
		{`title: "Test Post"`, `content: "Some content"`},
		{`content: "Some content"`, `author: "Someone"`},
		{`title: "Test Post"`, `author: "Someone"`},
	}
	for _, c := range cases {
		status, got, err := driver.CreatePost(c...)
		if err != nil {
			t.Fatalf("failed specification test, %v", err)
		}
		assertHTTPStatus(t, status, http.StatusBadRequest)
		want := map[string]any{
			"name":    httpserver.ErrMissingPostFields.Name,
			"message": httpserver.ErrMissingPostFields.Message,
		}
		v := decodeByJSON(t, got)
		json, _ := v.(map[string]any)
		assertJSONHasError(t, json)
		e, ok := json["error"].(map[string]any)
		if !ok {
			t.Fatal("didn't get expected error type")
		}
		assertGotError(t, e, want)
	}
}

func SuccessfullyReadPost(t testing.TB, driver ReadingPostAction) {
	status, got, err := driver.ReadPost(keepingPostId)
	if err != nil {
		t.Errorf("failed specification test, %v", err)
	}
	assertHTTPStatus(t, status, http.StatusOK)
	want := map[string]any{
		"id":      keepingPostId,
		"title":   "Test Post",
		"content": "Some content",
		"author":  "Someone",
	}
	v := decodeByJSON(t, got)
	json, _ := v.(map[string]any)
	assertJSONHasNoError(t, json)
	assertJSONHasData(t, json)
	data, ok := json["data"].(map[string]any)
	if !ok {
		t.Fatalf("didn't get expected data type")
	}
	assertPostsCanBeTheSame(t, data, want)
}

func decodeByJSON(t testing.TB, got string) any {
	t.Helper()

	var v any
	if err := json.NewDecoder(strings.NewReader(got)).Decode(&v); err != nil {
		t.Fatalf("unable to decode response payload")
	}
	return v
}

func assertHTTPStatus(t testing.TB, got, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("got status %d, but want %d", got, want)
	}
}

func assertJSONHasError(t testing.TB, got map[string]any) {
	t.Helper()

	if _, ok := got["error"]; !ok {
		t.Fatal("didn't get error")
	}
}

func assertJSONHasNoError(t testing.TB, got map[string]any) {
	t.Helper()

	if err, ok := got["error"]; ok {
		t.Fatalf("expected no error, but got %v", err)
	}
}

func assertJSONHasData(t testing.TB, got map[string]any) {
	t.Helper()

	if _, ok := got["data"]; !ok {
		t.Fatalf("didn't get data")
	}
}

func b_in_a(a, b map[string]any) bool {

	for ak, av := range a {
		bv, ok := b[ak]
		if !ok {
			continue
		}
		if av, ok := av.(map[string]any); ok {
			if bv, ok := bv.(map[string]any); ok {
				if b_in_a(av, bv) {
					continue
				}
				return false
			}
			return false
		}
		if !reflect.DeepEqual(av, bv) {
			return false
		}
	}
	return true
}

func assertPostsCanBeTheSame(t testing.TB, got, want map[string]any) {
	t.Helper()

	fn := func(p map[string]any) string {
		return fmt.Sprintf("{title=%s, content=%s, author=%s}", p["title"], p["content"], p["author"])
	}

	if !b_in_a(got, want) {
		t.Fatalf("got post %s, but want %s", fn(got), fn(want))
	}
}

func assertGotError(t testing.TB, got, want map[string]any) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got error %v, but want %v", got, want)
	}
}
