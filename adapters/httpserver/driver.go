package httpserver

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Args map[string]any

func (a Args) Get(names ...string) map[string]any {
	res := make(map[string]any)
	for _, name := range names {
		if v, ok := a[name]; ok {
			res[name] = v
		}
	}
	return res
}

func ParseArgs(args ...string) Args {
	res := make(Args)
	for _, arg := range args {
		splited := strings.SplitN(arg, ":", 2)
		if len(splited) != 2 {
			panic("argument must be `name: value` pattern")
		}
		name := strings.Trim(splited[0], " ")
		value := strings.Trim(splited[1], " ")
		res[name] = value
	}
	return res
}

type Driver struct {
	BaseURL string
	Client  *http.Client
	cookies []*http.Cookie
}

func (d *Driver) BeConnected(args ...string) error {

	body := createJSONBody(ParseArgs(args...))

	status, headers, _, err := d.doPost(d.BaseURL+"/login", "json", body)

	if err != nil {
		return err
	}

	if status != http.StatusOK {
		return fmt.Errorf("got status %d, but want %d", status, http.StatusOK)
	}

	if setCookie, ok := headers["Set-Cookie"]; ok {
		d.cookies = []*http.Cookie{}
		for _, setCookieInfo := range setCookie {
			cookie := &http.Cookie{}
			pairs := strings.Split(setCookieInfo, "; ")
			kv := strings.Split(pairs[0], "=")
			cookie.Name, cookie.Value = kv[0], kv[1]
			for _, pair := range pairs[1:] {
				kv := strings.Split(pair, "=")
				switch kv[0] {
				case "Max-Age":
					cookie.MaxAge, _ = strconv.Atoi(kv[1])
				case "HttpOnly":
					cookie.HttpOnly = true
				case "Path":
					cookie.Path = kv[1]
				case "Expires":
					cookie.Expires, _ = time.Parse(time.RFC1123, kv[1])
				}
			}
			d.cookies = append(d.cookies, cookie)
		}

		return nil
	}

	return errors.New("it's not connected")
}

func (d *Driver) CreateCustomer(args ...string) (int, string, error) {

	body := createJSONBody(ParseArgs(args...))

	status, _, _body, err := d.doPost(d.BaseURL+"/customers", "json", body)

	return status, _body, err
}

func (d *Driver) CreatePost(args ...string) (int, string, error) {

	body := createJSONBody(ParseArgs(args...))

	status, _, _body, err := d.doPost(d.BaseURL+"/posts", "json", body)

	return status, _body, err
}

func (d *Driver) ReadPost(id string) (int, string, error) {

	status, _, _body, err := d.doGet(d.BaseURL + "/posts/" + id)

	return status, _body, err
}

func createJSONBody(args Args) io.Reader {
	var body io.Reader
	if len(args) > 0 {
		builder := strings.Builder{}
		for name, value := range args {
			builder.WriteString(fmt.Sprintf(`"%s": %s,`, name, value))
		}
		body = strings.NewReader("{" + builder.String()[:builder.Len()-1] + "}") // removing the last comma
	} else {
		body = strings.NewReader("{}")
	}
	return body
}

func (d *Driver) handleWithResponse(res *http.Response) (int, http.Header, string, error) {
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, "", err
	}
	return res.StatusCode, res.Header, string(data), nil
}

func (d *Driver) consumeCookies(req *http.Request) {
	for _, cookie := range d.cookies {
		req.AddCookie(cookie)
	}
	clear(d.cookies)
	d.cookies = []*http.Cookie{}
}

func (d *Driver) doPost(url, contentType string, body io.Reader) (int, http.Header, string, error) {
	req, _ := http.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", contentType)
	d.consumeCookies(req)
	res, err := d.Client.Do(req)
	if err != nil {
		return 0, nil, "", err
	}
	return d.handleWithResponse(res)
}

func (d *Driver) doGet(url string) (int, http.Header, string, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	d.consumeCookies(req)
	res, err := d.Client.Do(req)
	if err != nil {
		return 0, nil, "", err
	}
	return d.handleWithResponse(res)
}
