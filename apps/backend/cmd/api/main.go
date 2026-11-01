package main

import (
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"kymber/backend/internal/logger"
	"kymber/backend/internal/routes"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		logger.Error("failed to load .env: %v", err)
		os.Exit(1)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	logger.Info("Kymber API started")
	logger.Info("Listening at http://localhost%s", addr)
	if err := http.ListenAndServe(addr, routes.New()); err != nil {
		logger.Error("server stopped: %v", err)
		os.Exit(1)
	}
}
