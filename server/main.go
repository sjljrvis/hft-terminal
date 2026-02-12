package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"hft/internal/brokers"
	"hft/internal/clock"
	"hft/internal/config"
	"hft/internal/executor"
	"hft/internal/storage/sqlite"
	"hft/server/routes"
)

func startAPIserver(mode string, cfg *config.Config, wsHub *routes.Hub) {
	addr := fmt.Sprintf(":%d", cfg.APIPort)
	log.Printf("API listening on %s", addr)
	if err := http.ListenAndServe(addr, routes.APIHandler(mode, cfg.DBPath, wsHub)); err != nil {
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
	modeFlag := flag.String("mode", "", "override mode: live|dryrun")
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
	if mode != "live" && mode != "dryrun" {
		log.Fatalf("invalid mode %q: must be 'live' or 'backtest'", mode)
	}

	log.Printf("starting server in %s mode (api_port=%d web_port=%d static=%s)", mode, cfg.APIPort, cfg.WebPort, *staticDir)

	// Initialize shared SQLite connection once (now returns *DB facade).
	sqlite.MustInitDefault(cfg.DBPath)

	brokers.Init()
	loginURL := brokers.LoginURL(cfg)
	log.Printf("login URL: %s", loginURL)

	// Initialize WebSocket hub for event streaming
	wsHub := routes.NewHub()
	go wsHub.Run()
	// Start event broadcaster to pipe executor events to WebSocket clients
	routes.StartEventBroadcaster(wsHub)
	log.Println("WebSocket event broadcaster started")

	// Start executor routine.
	exec := executor.NewExecutor(mode)
	go exec.Run()

	// Init clock + listen clock events.
	ctx := context.Background()
	mc, err := clock.NewMarketClockFromStrings(cfg.Clock.Location, cfg.Clock.Start, cfg.Clock.End, cfg.Clock.Deactivate)
	if err != nil {
		log.Printf("clock config error: %v (using defaults)", err)
		mc = clock.NewISTMarketClock()
	}

	// Debug: print the effective session boundaries in the clock timezone.
	now := time.Now().In(mc.Loc)
	y, m, d := now.Date()
	open := time.Date(y, m, d, mc.OpenHour, mc.OpenMin, 0, 0, mc.Loc)
	closeT := time.Date(y, m, d, mc.CloseHour, mc.CloseMin, 0, 0, mc.Loc)
	deactivate := time.Date(y, m, d, mc.DeactivateHour, mc.DeactivateMin, 0, 0, mc.Loc)
	log.Printf("clock: configured loc=%s start=%02d:%02d end=%02d:%02d deactivate=%02d:%02d | now=%s open=%s close=%s deactivate=%s",
		mc.Loc,
		mc.OpenHour, mc.OpenMin,
		mc.CloseHour, mc.CloseMin,
		mc.DeactivateHour, mc.DeactivateMin,
		now.Format(time.RFC3339),
		open.Format(time.RFC3339),
		closeT.Format(time.RFC3339),
		deactivate.Format(time.RFC3339),
	)

	openC, minuteC := mc.Start(ctx)

	go func() {
		for {
			select {
			case t, ok := <-openC:
				if !ok {
					return
				}
				log.Printf("clock: OPEN %s", t.In(mc.Loc).Format(time.RFC3339))
			case t, ok := <-minuteC:
				if !ok {
					return
				}
				log.Printf("clock: MINUTE %s", t.In(mc.Loc).Format(time.RFC3339))
			}
		}
	}()

	// Start API server.
	go startAPIserver(mode, cfg, wsHub)

	// Start webapp server (serves static assets if present, otherwise fallback text).
	go startWebserver(mode, *staticDir, cfg)

	select {} // block forever; servers/executor run in goroutines
}
