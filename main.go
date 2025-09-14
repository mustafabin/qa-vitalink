package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"vitalink/internal/models"
	"vitalink/internal/server"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { log.Fatal("DATABASE_URL is required") }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { log.Fatal(err) }

	if err := db.AutoMigrate(&models.PaymentPage{}); err != nil { log.Fatal(err) }

	e := server.Router(db)

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080" // Default port
	}

	if err := e.Start(":" + serverPort); err != nil { log.Fatal(err) }
}
