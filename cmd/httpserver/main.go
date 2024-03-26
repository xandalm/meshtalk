package main

import (
	"log"
	"meshtalk"
	"meshtalk/domain/services/memory"
	"net/http"
)

func main() {
	storage := memory.NewStorage()
	server := meshtalk.NewServer(storage)
	log.Fatal(http.ListenAndServe(":5000", server))
}
