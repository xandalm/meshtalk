package main

import (
	"log"
	"meshtalk"
	"net/http"
)

func main() {
	storage := meshtalk.NewInMemoryStorage()
	server := meshtalk.NewServer(storage)
	log.Fatal(http.ListenAndServe(":5000", server))
}
