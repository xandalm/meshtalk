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

			req, err := http.NewRequest(http.MethodGet, "http://dummy.site"+c.path, nil)

			if err != nil {
				t.Fatalf("unable to create http request, %v", err)
			}

			checkRouterEntries(t, router, c.want, req)
		})
	}
}

type SpyRequestParams struct {
	params map[string]string
}

func TestRouterParams(t *testing.T) {

	cases := []struct {
		pattern string
		path    string
		want    map[string]string
	}{
		{
			"/user/{id}",
			"/user/1",
			map[string]string{
				"id": "1",
			},
		},
		{
			"/org/{oid}/member/{mid}",
			"/org/1/member/11",
			map[string]string{
				"oid": "1",
				"mid": "11",
			},
		},
	}
	var spy *SpyRequestParams

	handle := RouteHandlerFunc(func(w http.ResponseWriter, r *Request) {
		spy.params = r.Params()
	})
	router := NewRouter()

	spy = &SpyRequestParams{}

	for _, c := range cases {
		t.Run(fmt.Sprintf(`add route to "%q", get with path "%q"`, c.pattern, c.path), func(t *testing.T) {
			router.UseFunc(c.pattern, handle)
			request, _ := http.NewRequest(http.MethodGet, "http://dummy.site"+c.path, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			checkParams(t, spy.params, c.want)
		})
	}
}

func checkRouterEntries(t *testing.T, router *Router, want routerEntry, request *http.Request) {
	t.Helper()

	got, ok := router.m[want.pattern]

	if !ok {
		t.Fatal("did not add pattern")
	}

	if (got.pattern != want.pattern) || (got.regexp.String() != want.regexp.String()) {
		t.Errorf(`got {pattern: "%s", regexp: %q}, want {pattern: "%s", regexp: %q}`, got.pattern, got.regexp.String(), want.pattern, want.regexp.String())
	}

	assertRouterEntryMatchesUrl(t, got, request)
}

func assertRouterEntryMatchesUrl(t testing.TB, entry routerEntry, request *http.Request) {
	t.Helper()

	if !entry.regexp.MatchString(request.URL.Path) {
		t.Fatalf(`did not match incoming "%s" url`, request.URL)
	}
}

func checkParams(t *testing.T, got, want map[string]string) {
	t.Helper()

	for key, value := range want {
		got, ok := got[key]

		if !ok {
			t.Fatalf(`expected %v to contain %s but it didn't`, got, key)
		}

		if got != value {
			t.Errorf(`expected %s equal to %q but got %q`, key, value, got)
		}
	}
}
