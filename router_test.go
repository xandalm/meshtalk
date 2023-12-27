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

func (h *StubRouterHandler) ServeHTTP(w meshtalk.ResponseWriter, r *meshtalk.Request) {
	h.recognized = append(h.recognized, r.URL.String())
	h.lastRecognizedParams = r.Params()
	h.numberOfCalls++
}

type testableURL struct {
	url                string
	expectedParams     params
	expectedHTTPStatus int
}

type keyValue map[string]string

type params keyValue

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
		{
			"/",
			[]testableURL{
				{
					url:                makeDummyHostUrl("", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusMovedPermanently,
				},
			},
		},
		{
			"/br/stores",
			[]testableURL{
				{
					url:                makeDummyHostUrl("/br/./stores", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusMovedPermanently,
				},
			},
		},
		{
			makeDummyHostUrl("/users", nil),
			[]testableURL{
				{
					url:                makeDummyHostUrl("/users", nil),
					expectedParams:     params{},
					expectedHTTPStatus: http.StatusOK,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf(`registering handler for "%s"`, c.pattern), func(t *testing.T) {
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

func TestRouterUse(t *testing.T) {
	handler := &StubRouterHandler{}
	router := meshtalk.NewRouter()

	cases := []struct {
		desc string
		p    string
		h    meshtalk.RouteHandler
		want string
	}{
		{"panic if try to register	empty pattern", "", handler, "router: invalid pattern"},
		{"panic if try to register nil handler", "/pattern", nil, "router: nil handler"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			defer func() {
				r := recover()
				str, ok := r.(string)
				if !ok || str != c.want {
					t.Errorf(`didn't panic %q`, c.want)
				}
			}()
			router.Use(c.p, c.h)
		})
	}

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

func TestRouterGet(t *testing.T) {

	router := meshtalk.NewRouter()
	router.Get("/users/{id}", &StubRouterHandler{})

	url := makeDummyHostUrl("/users/1", nil)

	t.Run("returns 200 on GET request", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodGet, url, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url, http.StatusOK)
	})

	otherMethodsCases := []string{
		http.MethodConnect,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodPatch,
	}

	for _, m := range otherMethodsCases {
		t.Run(fmt.Sprintf("returns 404 on %s request", m), func(t *testing.T) {
			request, _ := http.NewRequest(m, url, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assertGotStatus(t, response, url, http.StatusNotFound)
		})
	}
}

func TestRouterPost(t *testing.T) {
	router := meshtalk.NewRouter()
	router.Post("/users", &StubRouterHandler{})

	url := makeDummyHostUrl("/users", nil)

	t.Run("returns 200 on POST request", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodPost, url, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url, http.StatusOK)
	})

	otherMethodsCases := []string{
		http.MethodConnect,
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodPatch,
	}

	for _, m := range otherMethodsCases {
		t.Run(fmt.Sprintf("returns 404 on %s request", m), func(t *testing.T) {
			request, _ := http.NewRequest(m, url, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assertGotStatus(t, response, url, http.StatusNotFound)
		})
	}
}

func TestRouterPut(t *testing.T) {
	router := meshtalk.NewRouter()
	router.Put("/users/{id}", &StubRouterHandler{})

	url := makeDummyHostUrl("/users/1", nil)

	t.Run("returns 200 on POST request", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodPut, url, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url, http.StatusOK)
	})

	otherMethodsCases := []string{
		http.MethodConnect,
		http.MethodGet,
		http.MethodPost,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodPatch,
	}

	for _, m := range otherMethodsCases {
		t.Run(fmt.Sprintf("returns 404 on %s request", m), func(t *testing.T) {
			request, _ := http.NewRequest(m, url, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assertGotStatus(t, response, url, http.StatusNotFound)
		})
	}
}

func TestRouterDelete(t *testing.T) {
	router := meshtalk.NewRouter()
	router.Delete("/users/{id}", &StubRouterHandler{})

	url := makeDummyHostUrl("/users/1", nil)

	t.Run("returns 200 on POST request", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodDelete, url, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url, http.StatusOK)
	})

	otherMethodsCases := []string{
		http.MethodConnect,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodPatch,
	}

	for _, m := range otherMethodsCases {
		t.Run(fmt.Sprintf("returns 404 on %s request", m), func(t *testing.T) {
			request, _ := http.NewRequest(m, url, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assertGotStatus(t, response, url, http.StatusNotFound)
		})
	}
}

func TestRequestBody(t *testing.T) {
	router := meshtalk.NewRouter()
	handler := meshtalk.RouteHandlerFunc(func(w meshtalk.ResponseWriter, r *meshtalk.Request) {
		var input struct {
			Name string
		}
		err := r.BodyIn(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	})
	router.Post("/users", handler)
	url := makeDummyHostUrl("/users", nil)

	t.Run(`parses body into struct`, func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(`{"Name": "Alex"}`))
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url, http.StatusOK)
	})

	t.Run(`not parses body into struct`, func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(`[]`))
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url, http.StatusBadRequest)
	})

	t.Run("parses body into string", func(t *testing.T) {
		router.Post("/fruits", meshtalk.RouteHandlerFunc(func(w meshtalk.ResponseWriter, r *meshtalk.Request) {
			var input string
			err := r.BodyIn(&input)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}))

		url := makeDummyHostUrl("/fruits", nil)

		request, _ := http.NewRequest(http.MethodPost, url, strings.NewReader("banana"))
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url, http.StatusOK)
	})
}

func checkRouterRoutes(t *testing.T, router *meshtalk.Router, handler *StubRouterHandler, urlsToCheck []testableURL) {
	t.Helper()

	for _, url := range urlsToCheck {
		request, _ := http.NewRequest(http.MethodGet, url.url, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertGotStatus(t, response, url.url, url.expectedHTTPStatus)
		checkParams(t, handler.lastRecognizedParams, url.expectedParams)
	}
}

func assertGotStatus(t *testing.T, response *httptest.ResponseRecorder, url string, status int) {
	t.Helper()

	if response.Code != status {
		t.Errorf("%q got status %d but want %d", url, response.Code, status)
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
