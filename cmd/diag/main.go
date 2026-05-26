package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load("../../.env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	tables := []string{"articles", "categories", "users", "latest_news"}
	for _, table := range tables {
		fmt.Printf("\n=== %s ===\n", table)
		rows, err := db.Raw(`
			SELECT column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?
			ORDER BY ordinal_position
		`, table).Rows()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		for rows.Next() {
			var col, dtype, nullable string
			rows.Scan(&col, &dtype, &nullable)
			fmt.Printf("  %s (%s, nullable=%s)\n", col, dtype, nullable)
		}
		rows.Close()

		var count int64
		db.Table(table).Count(&count)
		fmt.Printf("  Row count: %d\n", count)
	}
}

