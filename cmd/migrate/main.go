package main

import (
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: go run cmd/migrate/main.go <migrations-path> <database-url> <command>")
	}

	migrationsPath := os.Args[1]
	databaseURL := os.Args[2]
	command := os.Args[3]

	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("An error occurred while migrating up: %v", err)
		}
		log.Println("Migrations applied successfully.")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("An error occurred while migrating down: %v", err)
		}
		log.Println("Migrations rolled back successfully.")
	default:
		log.Fatalf("Unknown command: %s. Use `up` or `down`.", command)
	}
}
