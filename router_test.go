package meshtalk

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
)

func TestRoutePatterns(t *testing.T) {
	dummyHandle := RouteHandlerFunc(func(w http.ResponseWriter, r *Request) {})
	cases := []struct {
		pattern string
		want    routerEntry
		path    string
	}{
		{
			"/user",
			routerEntry{
				dummyHandle,
				"/user",
				*regexp.MustCompile(`^\/user$`),
			},
			"/user",
		},
		{
			"/user/{id}",
			routerEntry{
				dummyHandle,
				"/user/{id}",
				*regexp.MustCompile(`^\/user\/\w+$`),
			},
			"/user/1",
		},
		{
			"/org/{id}/member/{id}",
			routerEntry{
				dummyHandle,
				"/org/{id}/member/{id}",
				*regexp.MustCompile(`^\/org\/\w+\/member\/\w+$`),
			},
			"/org/e6af1/member/1f276aeeab026521d532c5d3f10dd428",
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf(`adding route handle for "%s"`, c.pattern), func(t *testing.T) {
			router := NewRouter()
			router.UseFunc(c.pattern, dummyHandle)

			got, ok := router.m[c.pattern]

			if !ok {
				t.Fatal("did not add pattern")
			}

			if (got.pattern != c.want.pattern) || (got.regexp.String() != c.want.regexp.String()) {
				t.Errorf(`got {pattern: "%s", regexp: %q}, want {pattern: "%s", regexp: %q}`, got.pattern, got.regexp.String(), c.want.pattern, c.want.regexp.String())
			}

			if !got.regexp.MatchString(c.path) {
				t.Fatalf(`did not match incoming "%s" url path`, c.path)
			}
		})
	}

}
