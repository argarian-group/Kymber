package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func OpenFromEnv() (*sql.DB, error) {
	host := envOr("DB_HOST", "127.0.0.1")
	port := envOr("DB_PORT", "3306")
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("invalid DB_PORT %q: must be a number from 1 to 65535", port)
	}
	user, password, name := os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME")
	if user == "" || password == "" || name == "" {
		return nil, fmt.Errorf("DB_USER, DB_PASSWORD, and DB_NAME must be set")
	}

	config := mysql.Config{
		User: user, Passwd: password, Net: "tcp",
		Addr: net.JoinHostPort(host, port), DBName: name,
		ParseTime: true, Loc: time.UTC, Timeout: 5 * time.Second,
		AllowNativePasswords: true,
		Params:               map[string]string{"charset": "utf8mb4"},
	}
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("configure database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to MariaDB: %w", err)
	}
	return db, nil
}

func Migrate(db *sql.DB) error {
	migrations, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("load migration files: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectMySQL, db, migrations)
	if err != nil {
		return fmt.Errorf("configure migrations: %w", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
