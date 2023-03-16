//go:build mage
// +build mage

package main

import (
	"fmt"
	"time"

	_ "github.com/joho/godotenv/autoload" // Load .env file automatically
	"github.com/magefile/mage/sh"
)

// Migrate database.
func Migrate() error {
	fmt.Println("Migrating database...")
	return sh.Run("go", "run", "./cmd/migrate/")
}

// Up runs the application in development mode (with database migrations).
func Up() error {
	fmt.Println("Starting application...")

	// Start database
	if err := sh.Run("docker-compose", "up", "-d"); err != nil {
		return err
	}
	// Wait for database to be ready
	time.Sleep(5 * time.Second)

	// Migrate database
	if err := Migrate(); err != nil {
		return err
	}

	// Start application
	return sh.Run("go", "run", "./cmd/api/")
}

// Down stops the application.
func Down() error {
	fmt.Println("Stopping application...")
	return sh.Run("docker-compose", "down", "--rmi=local", "--volumes", "--remove-orphans")
}

// Run api
func Api() error {
	fmt.Println("Starting api...")
	return sh.Run("go", "run", "./cmd/api/")
}
