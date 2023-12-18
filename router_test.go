package meshtalk_test

import (
	"fmt"
	"meshtalk"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type StubRouterHandler struct {
	recognized    []string
	numberOfCalls int
}

func (h *StubRouterHandler) ServeHTTP(w http.ResponseWriter, r *meshtalk.Request) {
	h.recognized = append(h.recognized, r.URL.String())
	h.numberOfCalls++
}

type keyValue map[string]string

const dummyHost = "http://dummy.site"

func makeDummyHostUrl(path string, query keyValue) string {
	u := strings.Builder{}
	u.WriteString(dummyHost)

	if path != "" {
		for _, pl := range strings.Split(path, "/")[1:] {
			u.WriteRune('/')
			if pl != "" {
				u.WriteString(url.PathEscape(pl))
			}
		}
	}

	if len(query) > 0 {
		u.WriteRune('?')
		for k, v := range query {
			u.WriteString(url.QueryEscape(k) + "=" + url.QueryEscape(v) + "&")
		}
		return u.String()[:u.Len()-1]
	}
	return u.String()
}

func TestRouterPatterns(t *testing.T) {

	cases := []struct {
		pattern string
		pass    []string
		nopass  []string
	}{
		{
			"/user",
			[]string{
				makeDummyHostUrl("/user", nil),
			},
			[]string{
				makeDummyHostUrl("/user/1", nil),
			},
		},
		{
			"/user/{id}",
			[]string{
				makeDummyHostUrl("/user/1", nil),
				makeDummyHostUrl("/user/{id}", nil),
			},
			[]string{
				makeDummyHostUrl("/user", nil),
			},
		},
		{
			"/org/{oid}/member/{mid}",
			[]string{
				makeDummyHostUrl("/org/e6af1/member/1f276aeeab026521d532c5d3f10dd428", nil),
			},
			[]string{
				makeDummyHostUrl("/org", nil),
			},
		},
		{
			"/storage/{id}",
			[]string{
				makeDummyHostUrl("/storage/20", keyValue{"take": "food"}),
			},
			[]string{
				makeDummyHostUrl("/storag", nil),
			},
		},
		{
			"/user/",
			[]string{
				makeDummyHostUrl("/user/", nil),
				makeDummyHostUrl("/user", nil),
			},
			[]string{
				makeDummyHostUrl("/user/1", nil),
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf(`adding handle for "%s"`, c.pattern), func(t *testing.T) {
			router := meshtalk.NewRouter()

			handler := &StubRouterHandler{}

			router.Use(c.pattern, handler)

			checkRouterHandleUrls(t, router, handler, c.pass)
			checkRouterNotHandleUrls(t, router, handler, c.nopass)
		})
	}

	t.Run(`dintinguish "/users/" and "/users" when both is added`, func(t *testing.T) {
		router := meshtalk.NewRouter()

		handlerOne := &StubRouterHandler{}
		handlerTwo := &StubRouterHandler{}

		router.Use("/users/", handlerOne)
		router.Use("/users", handlerTwo)

		requestOne, _ := http.NewRequest(http.MethodGet, makeDummyHostUrl("/users/", nil), nil)
		responseOne := httptest.NewRecorder()

		requestTwo, _ := http.NewRequest(http.MethodGet, makeDummyHostUrl("/users", nil), nil)
		responseTwo := httptest.NewRecorder()

		router.ServeHTTP(responseOne, requestOne)
		router.ServeHTTP(responseTwo, requestTwo)

		assertStatus(t, responseOne, http.StatusOK)
		assertStatus(t, responseTwo, http.StatusOK)

		if handlerOne.numberOfCalls != 1 {
			t.Errorf("didn't call handler from /users/")
		}
		if handlerTwo.numberOfCalls != 1 {
			t.Errorf("didn't call handler from /users")
		}
	})
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

	handler := meshtalk.RouteHandlerFunc(func(w http.ResponseWriter, r *meshtalk.Request) {
		spy.params = r.Params()
	})
	router := meshtalk.NewRouter()

	spy = &SpyRequestParams{}

	for _, c := range cases {
		t.Run(fmt.Sprintf(`add route to "%q", get with path "%q"`, c.pattern, c.path), func(t *testing.T) {
			router.UseFunc(c.pattern, handler)
			request, _ := http.NewRequest(http.MethodGet, "http://dummy.site"+c.path, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			checkParams(t, spy.params, c.want)
		})
	}
}

func checkRouterHandleUrls(t *testing.T, router *meshtalk.Router, handler *StubRouterHandler, urls []string) {
	t.Helper()

	for _, url := range urls {
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("unable to create http request, %v", err)
		}
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertRouterHandle(t, handler, url, true)
	}
}

func checkRouterNotHandleUrls(t *testing.T, router *meshtalk.Router, handler *StubRouterHandler, urls []string) {
	t.Helper()

	for _, url := range urls {
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("unable to create http request, %v", err)
		}
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertRouterHandle(t, handler, url, false)
	}
}

func assertRouterHandle(t testing.TB, handler *StubRouterHandler, url string, expect bool) {
	t.Helper()

	contains := false
	for _, n := range handler.recognized {
		if n == url {
			contains = true
			break
		}
	}
	if expect && expect != contains {
		t.Errorf("expected %v to contain %q but it didn't", handler.recognized, url)
		return
	}
	if !expect && expect != contains {
		t.Errorf("expected %v to not contain %q but it did", handler.recognized, url)
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
