package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"hft/internal/backtest"
	"hft/internal/config"
	"hft/internal/execution"
	"hft/internal/storage/sqlite"
	"hft/server/routes"
)

func startAPIserver(mode string, cfg *config.Config) {
	addr := fmt.Sprintf(":%d", cfg.APIPort)
	log.Printf("API listening on %s", addr)
	if err := http.ListenAndServe(addr, routes.APIHandler(mode, cfg.DBPath)); err != nil {
		log.Fatalf("API server error: %v", err)
	}
}

func startWebserver(mode string, staticDir string, cfg *config.Config) {
	addr := fmt.Sprintf(":%d", cfg.WebPort)
	log.Printf("Webapp listening on %s (static dir: %s)", addr, staticDir)
	if err := http.ListenAndServe(addr, routes.WebHandler(mode, staticDir)); err != nil {
		log.Fatalf("Web server error: %v", err)
	}
}

func main() {
	configPath := flag.String("config", "configs/dev.yaml", "path to YAML config")
	modeFlag := flag.String("mode", "", "override mode: live|backtest")
	staticDir := flag.String("static-dir", "web-ui/dist", "path to built web assets")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mode := cfg.Mode
	if *modeFlag != "" {
		mode = *modeFlag
	}
	if mode != "live" && mode != "backtest" {
		log.Fatalf("invalid mode %q: must be 'live' or 'backtest'", mode)
	}

	log.Printf("starting server in %s mode (api_port=%d web_port=%d static=%s)", mode, cfg.APIPort, cfg.WebPort, *staticDir)

	// Initialize shared SQLite connection once (now returns *DB facade).
	sqlite.MustInitDefault(cfg.DBPath)

	backtest.Run()

	// Start executor routine.
	exec := execution.NewExecutor(mode)
	go exec.Run()

	// Start API server.
	go startAPIserver(mode, cfg)

	// Start webapp server (serves static assets if present, otherwise fallback text).
	go startWebserver(mode, *staticDir, cfg)

	select {} // block forever; servers/executor run in goroutines
}
