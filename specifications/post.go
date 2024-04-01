package specifications

import (
	"encoding/json"
	"fmt"
	"meshtalk/adapters/httpserver"
	"reflect"
	"strings"
	"testing"
)

type CreatePostAction interface {
	CreateAPost(args ...string) (string, error)
}

func SuccessfullyCreatePost(t testing.TB, driver CreatePostAction) {
	got, err := driver.CreateAPost(`title: "Test Post"`, `content: "Some content"`, `author: "Someone"`)
	if err != nil {
		t.Errorf("failed specification test, %v", err)
	}
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
		got, err := driver.CreateAPost(c...)
		if err != nil {
			t.Errorf("failed specification test, %v", err)
		}
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

func decodeByJSON(t testing.TB, got string) any {
	t.Helper()

	var v any
	if err := json.NewDecoder(strings.NewReader(got)).Decode(&v); err != nil {
		t.Fatalf("unable to decode response payload")
	}
	return v
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
