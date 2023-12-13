package meshtalk

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type routerEntry struct {
	h       RouteHandler
	pattern string
	regexp  regexp.Regexp
}

type RouteHandler interface {
	ServeHTTP(http.ResponseWriter, *Request)
}

type RouteHandlerFunc func(http.ResponseWriter, *Request)

func (f RouteHandlerFunc) ServeHTTP(w http.ResponseWriter, r *Request) {
	f(w, r)
}

type Request struct {
	params map[string]string
	query  map[string]string
	*http.Request
}

func (ro *Request) Params() map[string]string {
	if ro.params == nil {
		ro.params = make(map[string]string)
	}
	return ro.params
}

func (ro *Request) Query() map[string]string {
	if ro.query == nil {
		ro.query = make(map[string]string)
	}
	return ro.query
}

type Router struct {
	mu sync.RWMutex
	m  map[string]routerEntry
}

func NewRouter() *Router { return new(Router) }

func (ro *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &Request{nil, nil, r}

	path := r.URL.Path

	t, ok := ro.m[path]
	if ok {
		t.h.ServeHTTP(w, req)
	}
}

func (ro *Router) Use(pattern string, handle RouteHandler) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	if ro.m == nil {
		ro.m = make(map[string]routerEntry)
	}

	declaredParamsSeeker := regexp.MustCompile(`\/\{\w+\}`)
	declaredParamsBoundary := declaredParamsSeeker.FindAllStringIndex(pattern, -1)

	if len(declaredParamsBoundary) > 0 {

		builder := strings.Builder{}

		builder.WriteRune('^')
		b := declaredParamsBoundary[0]
		i := 0
		builder.WriteString(pattern[i:b[0]])
		builder.WriteString(`/\w+`)
		for _, boundary := range declaredParamsBoundary[1:] {
			i = b[1]
			b = boundary
			builder.WriteString(pattern[i:b[0]])
			builder.WriteString(`/\w+`)
		}

		builder.WriteString(pattern[b[1]:] + "$")

		ro.m[pattern] = routerEntry{
			handle,
			pattern,
			*regexp.MustCompile(strings.ReplaceAll(builder.String(), "/", `\/`)),
		}
		return
	}

	ro.m[pattern] = routerEntry{
		handle,
		pattern,
		*regexp.MustCompile("^" + strings.ReplaceAll(pattern, "/", `\/`) + "$"),
	}
}

func (ro *Router) UseFunc(pattern string, handler func(w http.ResponseWriter, r *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	ro.Use(pattern, RouteHandlerFunc(handler))
}
