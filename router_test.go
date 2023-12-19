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
	recognized           []string
	lastRecognizedParams params
	numberOfCalls        int
}

func (h *StubRouterHandler) ServeHTTP(w http.ResponseWriter, r *meshtalk.Request) {
	h.recognized = append(h.recognized, r.URL.String())
	h.lastRecognizedParams = r.Params()
	h.numberOfCalls++
}

type keyValue map[string]string

type params keyValue
type query keyValue

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

func TestRouterUse(t *testing.T) {
	handler := &StubRouterHandler{}
	router := meshtalk.NewRouter()
	t.Run("panic if try to register	empty pattern", func(t *testing.T) {
		defer func() {
			r := recover()
			str, ok := r.(string)
			if !ok || str != "router: invalid pattern" {
				t.Error(`didn't panic "router: invalid pattern"`)
			}
		}()
		router.Use("", handler)
	})
	t.Run("panic if try to register nil handler", func(t *testing.T) {
		defer func() {
			r := recover()
			str, ok := r.(string)
			if !ok || str != "router: nil handler" {
				t.Error(`didn't panic "router: nil handler"`)
			}
		}()
		router.Use("/pattern", nil)
	})
	t.Run("panic if try to register pattern again", func(t *testing.T) {
		defer func() {
			r := recover()
			str, ok := r.(string)
			if !ok || str != "router: multiple registration for /pattern" {
				t.Error(`didn't panic "router: multiple registration for /pattern"`)
			}
		}()
		router.Use("/pattern", handler)
		router.Use("/pattern", handler)
	})

	t.Run("usefunc panic if try to register nil handler", func(t *testing.T) {
		defer func() {
			r := recover()
			str, ok := r.(string)
			if !ok || str != "router: nil handler" {
				t.Error(`didn't panic "router: nil handler"`)
			}
		}()
		router.UseFunc("/pattern", nil)
	})
}

type testableURL struct {
	url                string
	expectedParams     params
	expectedHTTPStatus int
}

func TestRouter(t *testing.T) {
	cases := []struct {
		pattern      string
		testableURLs []testableURL
	}{
		{
			"/user",
			[]testableURL{
				{
					url:                makeDummyHostUrl("/user", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusOK,
				},
				{
					url:                makeDummyHostUrl("/user/1", nil),
					expectedHTTPStatus: http.StatusNotFound,
				},
			},
		},
		{
			"/user/{id}",
			[]testableURL{
				{
					url: makeDummyHostUrl("/user/1", nil),
					expectedParams: params{
						"id": "1",
					},
					expectedHTTPStatus: http.StatusOK,
				},
				{
					url: makeDummyHostUrl("/user/{id}", nil),
					expectedParams: params{
						"id": "{id}",
					},
					expectedHTTPStatus: http.StatusOK,
				},
				{
					url:                makeDummyHostUrl("/user/", nil),
					expectedHTTPStatus: http.StatusNotFound,
				},
			},
		},
		{
			"/org/{oid}/member/{mid}",
			[]testableURL{
				{
					url: makeDummyHostUrl("/org/e6af1/member/1f276aeeab026521d532c5d3f10dd428", nil),
					expectedParams: params{
						"oid": "e6af1",
						"mid": "1f276aeeab026521d532c5d3f10dd428",
					},
					expectedHTTPStatus: http.StatusOK,
				},
				{
					url:                makeDummyHostUrl("/org/12", nil),
					expectedHTTPStatus: http.StatusNotFound,
				},
			},
		},
		{
			"/storage/{id}",
			[]testableURL{
				{
					url: makeDummyHostUrl("/storage/20", keyValue{"take": "food"}),
					expectedParams: params{
						"id": "20",
					},
					expectedHTTPStatus: http.StatusOK,
				},
			},
		},
		{
			"/user/",
			[]testableURL{
				{
					url:                makeDummyHostUrl("/user/", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusOK,
				},
				{
					url:                makeDummyHostUrl("/user", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusMovedPermanently,
				},
			},
		},
		{
			"/user/{id}/",
			[]testableURL{
				{
					url:                makeDummyHostUrl("/user/1/", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusOK,
				},
				{
					url:                makeDummyHostUrl("/user/1", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusMovedPermanently,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf(`registering router handler for "%s"`, c.pattern), func(t *testing.T) {
			router := meshtalk.NewRouter()

			handler := &StubRouterHandler{}

			router.Use(c.pattern, handler)

			checkRouterRoutes(t, router, handler, c.testableURLs)
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
			t.Fatalf("didn't call handler from /users/")
		}
		if handlerTwo.numberOfCalls != 1 {
			t.Errorf("didn't call handler from /users")
		}
	})
}

func checkRouterRoutes(t *testing.T, router *meshtalk.Router, handler *StubRouterHandler, urlsToCheck []testableURL) {
	t.Helper()

	for _, url := range urlsToCheck {
		request, _ := http.NewRequest(http.MethodGet, url.url, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url)
		// assertRouterHandle(t, handler, url.url, true)
		checkParams(t, handler.lastRecognizedParams, url.expectedParams)
	}
}

func assertGotStatus(t *testing.T, response *httptest.ResponseRecorder, url testableURL) {
	t.Helper()

	if response.Code != url.expectedHTTPStatus {
		t.Errorf("%q got status %d but want %d", url.url, response.Code, url.expectedHTTPStatus)
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
		t.Fatalf("expected %v to contain %q but it didn't", handler.recognized, url)
		return
	}
	if !expect && expect != contains {
		t.Fatalf("expected %v to not contain %q but it did", handler.recognized, url)
	}
}

func checkParams(t *testing.T, got, want map[string]string) {
	t.Helper()

	if want == nil {
		return
	}

	for key, value := range want {
		gotValue, ok := got[key]

		if !ok || gotValue != value {
			t.Fatalf(`expected %v to contain %s equal to %q but it didn't`, got, key, value)
		}
	}
}
