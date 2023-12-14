package meshtalk

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRouterPatterns(t *testing.T) {
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
				*regexp.MustCompile(`^\/user\/(?P<id>\w+)$`),
			},
			"/user/1",
		},
		{
			"/org/{oid}/member/{mid}",
			routerEntry{
				dummyHandle,
				"/org/{oid}/member/{mid}",
				*regexp.MustCompile(`^\/org\/(?P<oid>\w+)\/member\/(?P<mid>\w+)$`),
			},
			"/org/e6af1/member/1f276aeeab026521d532c5d3f10dd428",
		},
		{
			"/storage/{id}",
			routerEntry{
				dummyHandle,
				"/storage/{id}",
				*regexp.MustCompile(`^\/storage\/(?P<id>\w+)$`),
			},
			"/storage/20?take=foods",
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf(`adding handle for "%s"`, c.pattern), func(t *testing.T) {
			router := NewRouter()
			router.UseFunc(c.pattern, dummyHandle)

			got, ok := router.m[c.pattern]

			if !ok {
				t.Fatal("did not add pattern")
			}

			if (got.pattern != c.want.pattern) || (got.regexp.String() != c.want.regexp.String()) {
				t.Errorf(`got {pattern: "%s", regexp: %q}, want {pattern: "%s", regexp: %q}`, got.pattern, got.regexp.String(), c.want.pattern, c.want.regexp.String())
			}

			req, _ := http.NewRequest(http.MethodGet, "http://dummy.site"+c.path, nil)

			if !got.regexp.MatchString(req.URL.Path) {
				t.Fatalf(`did not match incoming "%s" url path`, c.path)
			}
		})
	}
}

type SpyRequestParams struct {
	params map[string]string
}

func TestRouterParams(t *testing.T) {

	spy := SpyRequestParams{}

	pattern := "/user/{id}"
	handle := RouteHandlerFunc(func(w http.ResponseWriter, r *Request) {
		spy.params = r.Params()
	})

	path := "/user/10"

	router := NewRouter()
	router.UseFunc(pattern, handle)

	request, _ := http.NewRequest(http.MethodGet, "http://dummy.site"+path, nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	got, ok := spy.params["id"]
	want := "10"

	if !ok {
		t.Errorf(`expected %v to contain "id" but it didn't`, spy.params)
	}

	if got != want {
		t.Errorf(`expected id equal to %q but got %q`, want, got)
	}
}
