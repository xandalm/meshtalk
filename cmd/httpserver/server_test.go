package main_test

import (
	"context"
	"log"
	"meshtalk/adapters/httpserver"
	"meshtalk/specifications"
	"net/http"
	"testing"
	"time"

	xtesting "github.com/xandalm/go-testing"
)

func TestServer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	var (
		baseURL = "http://localhost:5000"
		client  = &http.Client{
			Timeout: 2 * time.Second,
		}
		driver = &httpserver.Driver{
			BaseURL: baseURL,
			Client:  client,
		}
	)

	launcher := xtesting.NewServerLauncher(context.Background(), "", "main.go", &xtesting.HTTPServerChecker{
		BaseURL: baseURL,
		Cli:     client,
	})

	if err := launcher.StartAndWait(2 * time.Second); err != nil {
		log.Fatalf("unable to launch the server, %v", err)
	}
	defer func() {
		if err := launcher.EndAndClean(); err != nil {
			log.Fatalf("unable to gracefully end the server, %v", err)
		}
	}()

	specifications.SuccessfullyCreatePost(t, driver)
	specifications.UnableToCreatePostDueToMissingRequiredValues(t, driver)
	specifications.SuccessfullyReadPost(t, driver)
}
