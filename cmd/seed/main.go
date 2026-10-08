package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/database/seeds"
)

func main() {
	tableFlag := flag.String("table", "all", "Specify table to seed ('all', 'hospital')")
	forceFlag := flag.Bool("force", false, "Force execution even in production environment")
	flag.Parse()

	cfg := config.Load()

	// Safety check: Prevent accidental execution in production environment
	isProduction := cfg.Server.Mode == "release" || os.Getenv("APP_ENV") == "production"
	if isProduction && !*forceFlag {
		log.Fatalf("[ABORT] Running database seeder in production is restricted! Use -force flag if intentional.")
	}

	log.Printf("[INFO] Connecting to database '%s' at %s:%s...", cfg.DB.DBName, cfg.DB.Host, cfg.DB.Port)
	db, err := database.New(cfg.DB)
	if err != nil {
		log.Fatalf("[ERROR] Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	log.Printf("[INFO] Executing seeder (target: %s)...", *tableFlag)

	switch *tableFlag {
	case "all":
		if err := seeds.RunAll(ctx, db); err != nil {
			log.Fatalf("[ERROR] Seeder execution failed: %v", err)
		}
	case "hospital":
		if err := seeds.SeedHospitals(ctx, db); err != nil {
			log.Fatalf("[ERROR] Hospital seeder failed: %v", err)
		}
	default:
		log.Fatalf("[ERROR] Unknown table target %q. Available targets: 'all', 'hospital'", *tableFlag)
	}

	duration := time.Since(start).Round(time.Millisecond)
	fmt.Printf("\n [SUCCESS] Database seeding completed successfully in %v!\n\n", duration)
}
