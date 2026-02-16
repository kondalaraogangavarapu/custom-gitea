package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aetherdev/aetherdev/internal/api"
	"github.com/aetherdev/aetherdev/internal/config"
)

var (
	Version   = "0.1.0"
	BuildDate = "dev"
)

func main() {
	configPath := flag.String("config", "aetherdev.yaml", "path to configuration file")
	port := flag.Int("port", 3000, "server port")
	dataDir := flag.String("data", "./data", "data directory for repositories and database")
	showVersion := flag.Bool("version", false, "show version info")
	flag.Parse()

	if *showVersion {
		fmt.Printf("AetherDev v%s (built %s)\n", Version, BuildDate)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Using default configuration: %v", err)
		cfg = config.Default()
	}

	if *port != 3000 {
		cfg.Server.Port = *port
	}
	if *dataDir != "./data" {
		cfg.DataDir = *dataDir
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.MkdirAll(cfg.RepoRootPath(), 0755); err != nil {
		log.Fatalf("Failed to create repositories directory: %v", err)
	}

	router, err := api.NewRouter(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize router: %v", err)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("AetherDev v%s starting on http://localhost:%d", Version, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down AetherDev...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}
	log.Println("AetherDev stopped.")
}
