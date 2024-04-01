package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"strings"
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
}

func (d *Driver) CreatePost(args ...string) (int, string, error) {
	_args := ParseArgs(args...)

	var body io.Reader
	if len(_args) > 0 {
		builder := strings.Builder{}
		for name, value := range _args {
			builder.WriteString(fmt.Sprintf(`"%s": %s,`, name, value))
		}
		body = strings.NewReader("{" + builder.String()[:builder.Len()-1] + "}") // removing the last comma
	} else {
		body = strings.NewReader("{}")
	}

	return d.doPost(d.BaseURL+"/posts", "json", body)
}

func (d *Driver) ReadPost(id string) (int, string, error) {
	return d.doGet(d.BaseURL + "/posts/" + id)
}

func (d *Driver) handleWithResponse(res *http.Response) (int, string, error) {
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, "", err
	}
	return res.StatusCode, string(data), nil
}

func (d *Driver) doPost(url, contentType string, body io.Reader) (int, string, error) {
	res, err := d.Client.Post(url, contentType, body)
	if err != nil {
		return 0, "", err
	}
	return d.handleWithResponse(res)
}

func (d *Driver) doGet(url string) (int, string, error) {
	res, err := d.Client.Get(url)
	if err != nil {
		return 0, "", err
	}
	return d.handleWithResponse(res)
}
