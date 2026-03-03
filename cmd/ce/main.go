package main

import (
	"log"
	"net/http"
	"os"

	"github.com/liyu1981/code_explorer/pkg/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := server.New()
	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatal(err)
	}
}
