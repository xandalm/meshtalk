package meshtalk

import (
	"net/http"
	"net/url"
	"path"
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
	ServeHTTP(ResponseWriter, *Request)
}

type RouteHandlerFunc func(ResponseWriter, *Request)

func (f RouteHandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

type redirectHandler struct {
	url  string
	code int
}

func (rh *redirectHandler) ServeHTTP(w ResponseWriter, r *Request) {
	http.Redirect(w, r.Request, rh.url, rh.code)
}

func RedirectHandler(url string, code int) RouteHandler {
	return &redirectHandler{url, code}
}

type ResponseWriter http.ResponseWriter

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
	mu   sync.RWMutex
	m    map[string]routerEntry
	sm   map[string]routerEntry
	host bool
}

func NewRouter() *Router { return new(Router) }

func (ro *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	req := &Request{nil, nil, r}

	_, h, params := ro.Handler(req)
	req.params = params
	h.ServeHTTP(w, req)
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}

	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)

	if p[len(p)-1] == '/' && np != "/" {
		if len(p) == len(np)+1 && strings.HasPrefix(p, np) {
			np = p
		} else {
			np += "/"
		}
	}

	return np
}

func (ro *Router) Handler(r *Request) (p string, h RouteHandler, params map[string]string) {

	host := r.URL.Host
	path := cleanPath(r.URL.Path)

	if ro.host {
		p, h, params = ro.match(host + path)
	}

	if h == nil {
		p, h, params = ro.match(path)
	}

	if h != nil {
		return
	}

	if ro.shouldRedirectToSlashPath(path) {
		u := &url.URL{Path: path + "/", RawQuery: r.URL.RawQuery}
		return u.Path, RedirectHandler(u.String(), http.StatusMovedPermanently), nil
	}

	return p, NotFoundHandler(), nil
}

func (ro *Router) shouldRedirectToSlashPath(path string) bool {
	ro.mu.RLock()
	defer ro.mu.RUnlock()

	path = path + "/"

	if _, ok := ro.sm[path]; ok {
		return true
	}

	for _, e := range ro.sm {
		if e.regexp.MatchString(path) {
			return true
		}
	}

	return false
}

func (ro *Router) match(path string) (string, RouteHandler, map[string]string) {
	ro.mu.RLock()
	defer ro.mu.RUnlock()
	// Exatly match
	e, ok := ro.m[path]
	if ok && e.regexp.MatchString(path) {
		matches := e.regexp.FindStringSubmatch(path)
		params := make(map[string]string)

		for i, tag := range e.regexp.SubexpNames() {
			if i != 0 && tag != "" {
				params[tag] = matches[i]
			}
		}
		return e.pattern, e.h, params
	}

	for _, e := range ro.m {
		if e.regexp.MatchString(path) {
			matches := e.regexp.FindStringSubmatch(path)
			params := make(map[string]string)

			for i, tag := range e.regexp.SubexpNames() {
				if i != 0 && tag != "" {
					params[tag] = matches[i]
				}
			}
			return e.pattern, e.h, params
		}
	}

	return "", nil, nil
}

func NotFoundHandler() RouteHandler {
	return RouteHandlerFunc(func(w ResponseWriter, r *Request) { http.Error(w, "404 page not found", http.StatusNotFound) })
}

func createRegExp(pattern string) *regexp.Regexp {

	builder := strings.Builder{}

	builder.WriteRune('^')

	paramsSeeker := regexp.MustCompile(`(\/\{[^\/]+\})`)
	builder.WriteString(paramsSeeker.ReplaceAllStringFunc(pattern, func(m string) string {
		return "/(?P<" + m[2:len(m)-1] + ">[^/]+)"
	}))

	builder.WriteString("$")

	return regexp.MustCompile(strings.ReplaceAll(builder.String(), "/", `\/`))
}

func (ro *Router) Use(pattern string, handler RouteHandler) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	pattern = regexp.MustCompile(`^https?\:\/\/`).ReplaceAllString(pattern, "")

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

	// paramsBound := findParamsBound(pattern)

	patternRegExp := createRegExp(pattern /* , paramsBound */)

	e := routerEntry{
		handler,
		pattern,
		*patternRegExp,
	}

	ro.m[pattern] = e

	if pattern[len(pattern)-1] == '/' {
		ro.registerSlashedEntry(e)
	}

	ro.host = pattern[0] != '/'

}

func (ro *Router) registerSlashedEntry(e routerEntry) {
	if ro.sm == nil {
		ro.sm = make(map[string]routerEntry)
	}

	ro.sm[e.pattern] = e
}

func (ro *Router) UseFunc(pattern string, handler func(w ResponseWriter, r *Request)) {
	if handler == nil {
		panic("router: nil handler")
	}
	ro.Use(pattern, RouteHandlerFunc(handler))
}
