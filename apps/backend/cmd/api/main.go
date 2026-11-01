package main

import (
	"os"

	"github.com/joho/godotenv"
	"kymber/backend/internal/logger"
	"kymber/backend/internal/server"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil && !os.IsNotExist(err) {
		logger.Error("failed to load .env: %v", err)
		os.Exit(1)
	}

	if err := server.Run(os.Getenv("SERVER_PORT")); err != nil {
		logger.Error("server stopped: %v", err)
		os.Exit(1)
	}
}
