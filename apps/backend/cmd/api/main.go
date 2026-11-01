package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"kymber/backend/internal/routes"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, routes.New()))
}
