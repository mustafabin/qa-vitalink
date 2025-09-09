package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"vitalink/internal/server"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { log.Fatal("DATABASE_URL is required") }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { log.Fatal(err) }

	e := server.Router(db)
	if err := e.Start(":8080"); err != nil { log.Fatal(err) }
}
