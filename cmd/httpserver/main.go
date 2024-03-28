package main

import (
	"log"
	"meshtalk/adapters/httpserver"
	"meshtalk/domain/services/memory"
	"net/http"
)

func main() {
	storage := memory.NewStorage()
	server := httpserver.NewServer(storage)
	log.Fatal(http.ListenAndServe(":5000", server))
}
