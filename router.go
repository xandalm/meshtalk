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

	_, h := ro.Handler(req)
	h.ServeHTTP(w, req)
}

func (ro *Router) Handler(r *Request) (pattern string, handler RouteHandler) {

	return ro.match(r)
}

func (ro *Router) match(r *Request) (string, RouteHandler) {

	path := r.URL.Path

	// Exatly match
	e, ok := ro.m[path]
	if ok && e.regexp.MatchString(path) {
		return e.pattern, e.h
	}

	for _, e := range ro.m {
		if e.regexp.MatchString(path) {
			matches := e.regexp.FindStringSubmatch(path)
			r.params = map[string]string{}

			for i, tag := range e.regexp.SubexpNames() {
				if i != 0 && tag != "" {
					r.params[tag] = matches[i]
				}
			}
			return e.pattern, e.h
		}
	}

	return "", NotFoundHandler()
}

func NotFoundHandler() RouteHandler {
	return RouteHandlerFunc(func(w http.ResponseWriter, r *Request) { http.Error(w, "404 page not found", http.StatusNotFound) })
}

func findParamsBound(pattern string) [][]int {
	paramsSeeker := regexp.MustCompile(`\/\{[^\/]+\}`)
	return paramsSeeker.FindAllStringIndex(pattern, -1)
}

func taggedParam(pattern string, s, e int) string {
	return `/(?P<` + pattern[(s+2):(e-1)] + `>[^\/]+)`
}

func createRegExp(pattern string, paramsBound [][]int) *regexp.Regexp {

	shouldMakeSlashOptional := pattern[len(pattern)-1] == '/'

	if len(paramsBound) > 0 {
		builder := strings.Builder{}

		builder.WriteRune('^')

		i := 0
		for _, b := range paramsBound {
			builder.WriteString(pattern[i:b[0]])
			builder.WriteString(taggedParam(pattern, b[0], b[1]))
			i = b[1]
		}
		if i < (len(pattern) - 1) {
			builder.WriteString(pattern[i:])
		}

		if shouldMakeSlashOptional {
			builder.WriteRune('?')
		}

		builder.WriteString("$")

		return regexp.MustCompile(strings.ReplaceAll(builder.String(), "/", `\/`))
	}

	if shouldMakeSlashOptional {
		return regexp.MustCompile("^" + strings.ReplaceAll(pattern, "/", `\/`) + "?$")
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

	e := routerEntry{
		handle,
		pattern,
		*patternRegExp,
	}

	ro.m[pattern] = e
}

func (ro *Router) UseFunc(pattern string, handler func(w http.ResponseWriter, r *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	ro.Use(pattern, RouteHandlerFunc(handler))
}
