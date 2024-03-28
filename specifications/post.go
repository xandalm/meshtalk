package specifications

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type CreatePostAction interface {
	CreateAPost(args ...string) (string, error)
}

func CreatingAPostSpecification(t testing.TB, driver CreatePostAction) {
	got, err := driver.CreateAPost("title: Test Post", "content: Some content", "author: Someone")
	if err != nil {
		t.Fatalf("failed specification test, %v", err)
	}
	want := map[string]any{
		"title":   "Test Post",
		"content": "Some content",
		"author":  "Someone",
	}
	var v any
	if err := json.NewDecoder(strings.NewReader(got)).Decode(&v); err != nil {
		t.Fatalf("unable to decode response payload")
	}
	json, _ := v.(map[string]any)
	assertJSONHasNoError(t, json)
	assertJSONHasData(t, json)
	data, ok := json["data"].(map[string]any)
	if !ok {
		t.Fatalf("didn't get expected data type")
	}
	assertPostsCanBeTheSame(t, data, want)
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
