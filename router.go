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
		return make(map[string]string)
	}
	return ro.params
}

func (ro *Request) Query() map[string]string {
	if ro.query == nil {
		return make(map[string]string)
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

	// Exatly match
	t, ok := ro.m[path]
	if ok {
		t.h.ServeHTTP(w, req)
		return
	}

	for _, e := range ro.m {
		if e.regexp.MatchString(path) {
			matches := e.regexp.FindStringSubmatch(path)
			req.params = map[string]string{}

			for i, tag := range e.regexp.SubexpNames() {
				if i != 0 && tag != "" {
					req.params[tag] = matches[i]
				}
			}
			e.h.ServeHTTP(w, req)
			break
		}
	}
}

func findParamsBound(pattern string) [][]int {
	paramsSeeker := regexp.MustCompile(`\/\{\w+\}`)
	return paramsSeeker.FindAllStringIndex(pattern, -1)
}

func createRegExp(pattern string, paramsBound [][]int) *regexp.Regexp {
	if len(paramsBound) > 0 {
		builder := strings.Builder{}

		builder.WriteRune('^')
		b := paramsBound[0]
		i := 0
		builder.WriteString(pattern[i:b[0]])
		builder.WriteString(`/(?P<` + pattern[(b[0]+2):(b[1]-1)] + `>\w+)`)
		for _, boundary := range paramsBound[1:] {
			i = b[1]
			b = boundary
			builder.WriteString(pattern[i:b[0]])
			builder.WriteString(`/(?P<` + pattern[(b[0]+2):(b[1]-1)] + `>\w+)`)
		}
		builder.WriteString(pattern[b[1]:] + "$")

		return regexp.MustCompile(strings.ReplaceAll(builder.String(), "/", `\/`))
	}

	return regexp.MustCompile("^" + strings.ReplaceAll(pattern, "/", `\/`) + "$")
}

func (ro *Router) Use(pattern string, handle RouteHandler) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	if ro.m == nil {
		ro.m = make(map[string]routerEntry)
	}

	paramsBound := findParamsBound(pattern)

	patternRegExp := createRegExp(pattern, paramsBound)

	ro.m[pattern] = routerEntry{
		handle,
		pattern,
		*patternRegExp,
	}
}

func (ro *Router) UseFunc(pattern string, handler func(w http.ResponseWriter, r *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	ro.Use(pattern, RouteHandlerFunc(handler))
}
