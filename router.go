package meshtalk

import (
	"net/http"
	"net/url"
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

type redirectHandler struct {
	url  string
	code int
}

func (rh *redirectHandler) ServeHTTP(w http.ResponseWriter, r *Request) {
	http.Redirect(w, r.Request, rh.url, rh.code)
}

func RedirectHandler(url string, code int) RouteHandler {
	return &redirectHandler{url, code}
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
	sm map[string]routerEntry
}

func NewRouter() *Router { return new(Router) }

func (ro *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	req := &Request{nil, nil, r}

	_, h := ro.Handler(req)
	h.ServeHTTP(w, req)
}

func (ro *Router) Handler(r *Request) (pattern string, handler RouteHandler) {
	ro.mu.RLock()
	defer ro.mu.RUnlock()

	p, h := ro.match(r)

	if h == nil {

		slashedPath := r.URL.Path + "/"
		var u *url.URL

		if _, ok := ro.sm[slashedPath]; ok {
			u = &url.URL{Path: slashedPath, RawQuery: r.URL.RawQuery}
		}

		if u == nil {
			for _, e := range ro.sm {
				if e.regexp.MatchString(slashedPath) {
					u = &url.URL{Path: slashedPath, RawQuery: r.URL.RawQuery}
					break
				}
			}
		}

		if u == nil {
			return p, NotFoundHandler()
		}

		return p, RedirectHandler(u.String(), http.StatusMovedPermanently)
	}

	return p, h
}

func (ro *Router) match(r *Request) (string, RouteHandler) {

	path := r.URL.Path

	// Exatly match
	e, ok := ro.m[path]
	if ok && e.regexp.MatchString(path) {
		matches := e.regexp.FindStringSubmatch(path)
		r.params = map[string]string{}

		for i, tag := range e.regexp.SubexpNames() {
			if i != 0 && tag != "" {
				r.params[tag] = matches[i]
			}
		}
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

	return "", nil
}

func NotFoundHandler() RouteHandler {
	return RouteHandlerFunc(func(w http.ResponseWriter, r *Request) { http.Error(w, "404 page not found", http.StatusNotFound) })
}

func findParamsBound(pattern string) [][]int {
	paramsSeeker := regexp.MustCompile(`\/\{[^\/]+\}`)
	return paramsSeeker.FindAllStringIndex(pattern, -1)
}

func taggedParam(pattern string, s, e int) string {
	return `/(?P<` + pattern[(s+2):(e-1)] + `>[^/]+)`
}

func createRegExp(pattern string, paramsBound [][]int) *regexp.Regexp {

	if len(paramsBound) > 0 {
		builder := strings.Builder{}

		builder.WriteRune('^')

		i := 0
		for _, b := range paramsBound {
			builder.WriteString(pattern[i:b[0]])
			builder.WriteString(taggedParam(pattern, b[0], b[1]))
			i = b[1]
		}
		if i < len(pattern) {
			builder.WriteString(pattern[i:])
		}

		builder.WriteString("$")

		return regexp.MustCompile(strings.ReplaceAll(builder.String(), "/", `\/`))
	}

	return regexp.MustCompile("^" + strings.ReplaceAll(pattern, "/", `\/`) + "$")
}

func (ro *Router) Use(pattern string, handler RouteHandler) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	if pattern == "" {
		panic("router: invalid pattern")
	}
	if handler == nil {
		panic("router: nil handler")
	}
	if _, ok := ro.m[pattern]; ok {
		panic("router: multiple registration for " + pattern)
	}

	if ro.m == nil {
		ro.m = make(map[string]routerEntry)
	}

	paramsBound := findParamsBound(pattern)

	patternRegExp := createRegExp(pattern, paramsBound)

	e := routerEntry{
		handler,
		pattern,
		*patternRegExp,
	}

	ro.m[pattern] = e

	if pattern[len(pattern)-1] == '/' {
		ro.registerSlashedEntry(e)
	}
}

func (ro *Router) registerSlashedEntry(e routerEntry) {
	if ro.sm == nil {
		ro.sm = make(map[string]routerEntry)
	}

	ro.sm[e.pattern] = e
}

func (ro *Router) UseFunc(pattern string, handler func(w http.ResponseWriter, r *Request)) {
	if handler == nil {
		panic("router: nil handler")
	}
	ro.Use(pattern, RouteHandlerFunc(handler))
}
