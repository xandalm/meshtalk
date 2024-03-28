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

func (d *Driver) CreateAPost(args ...string) (string, error) {
	requiredArgs := ParseArgs(args...).Get("title", "content", "author")
	if len(requiredArgs) != 3 {
		return "", fmt.Errorf("missing required args")
	}
	title, _ := requiredArgs["title"].(string)
	content, _ := requiredArgs["content"].(string)
	author, _ := requiredArgs["author"].(string)

	body := strings.NewReader(fmt.Sprintf(`{"title": "%s", "content": "%s", "author": "%s"}`,
		title,
		content,
		author))

	res, err := d.Client.Post(d.BaseURL+"/posts", "json", body)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
