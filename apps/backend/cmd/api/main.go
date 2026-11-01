package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"kymber/backend/internal/database"
	"kymber/backend/internal/logger"
	"kymber/backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		logger.Error("application stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	if err := godotenv.Load("../../.env"); err != nil && !os.IsNotExist(err) {
		return err
	}
	secureOnboard := false
	if value := os.Getenv("SECURE_ONBOARD"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid SECURE_ONBOARD %q: must be true or false", value)
		}
		secureOnboard = parsed
	}

	db, err := database.OpenFromEnv()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		return err
	}

	return server.Run(os.Getenv("SERVER_PORT"), db, secureOnboard)
}
