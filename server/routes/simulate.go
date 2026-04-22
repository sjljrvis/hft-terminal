package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"hft/internal/executor"
)

// SimulateHandler handles POST /simulate to replay a past date bar-by-bar.
//
//	POST /simulate {"date": "2026-03-13", "warmupDays": 100, "tickDelay": 10}
//
// tickDelay is seconds between bars (default 10). Runs in background;
// events stream to /ws/events WebSocket clients.
func SimulateHandler(wsHub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Date       string `json:"date"`
			WarmupDays int    `json:"warmupDays"`
			TickDelay  int    `json:"tickDelay"` // seconds
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		if req.Date == "" {
			http.Error(w, "date is required (e.g. 2026-03-13)", http.StatusBadRequest)
			return
		}
		if req.WarmupDays <= 0 {
			req.WarmupDays = 100
		}
		if req.TickDelay <= 0 {
			req.TickDelay = 10
		}

		cfg := executor.SimConfig{
			SimDate:    req.Date,
			WarmupDays: req.WarmupDays,
			TickDelay:  time.Duration(req.TickDelay) * time.Second,
			OnEvent: func(eventType string, data map[string]interface{}) {
				wsHub.BroadcastMessage(eventType, data)
			},
		}

		// Run simulation in background — events stream to WS clients.
		go func() {
			if err := executor.RunSimulation(cfg); err != nil {
				log.Printf("simulate: error: %v", err)
				wsHub.BroadcastMessage("sim_error", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "started",
			"date":    req.Date,
			"message": fmt.Sprintf("Simulation started for %s. Events streaming to /ws/events.", req.Date),
		})
	}
}
