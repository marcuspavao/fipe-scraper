package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marcuspavao/fipe-scraper/internal/db"
	"github.com/marcuspavao/fipe-scraper/internal/migrations"
)

func main() {
	cmd := flag.String("cmd", "up", "Comando: up | down | status")
	flag.Parse()

	log.Println("🗄️  FIPE Migrate iniciando...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		log.Fatalf("❌ MongoDB: %v", err)
	}
	defer db.Disconnect()
	log.Println("✅ MongoDB conectado")

	database := db.GetDatabase()

	switch *cmd {
	case "up":
		log.Println("⬆️  Aplicando migrations pendentes...")
		if err := migrations.Run(context.Background(), database); err != nil {
			log.Fatalf("❌ %v", err)
		}

	case "down":
		log.Println("⬇️  Revertendo última migration...")
		if err := migrations.Rollback(context.Background(), database); err != nil {
			log.Fatalf("❌ %v", err)
		}

	case "status":
		if err := migrations.Status(context.Background(), database); err != nil {
			log.Fatalf("❌ %v", err)
		}

	default:
		fmt.Fprintf(os.Stderr, "❌ Comando inválido: %q\n", *cmd)
		fmt.Fprintln(os.Stderr, "   Use: -cmd up | -cmd down | -cmd status")
		os.Exit(1)
	}

	os.Exit(0)
}
