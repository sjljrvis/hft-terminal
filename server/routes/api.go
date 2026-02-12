package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"hft/internal/executor"
	"hft/internal/storage/sqlite"
)

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// APIHandler exposes the API HTTP handler.
func APIHandler(mode string, dbPath string, wsHub *Hub) http.Handler {
	mux := http.NewServeMux()

	// General endpoints
	mux.HandleFunc("/hft/status", HFTStatusHandler)
	mux.HandleFunc("/db/query", DBQueryHandler(dbPath))

	mux.HandleFunc("/broker/fyers/callback", FyersLoginHandler)
	mux.HandleFunc("/broker/fyers/margin", FyersMarginHandler) // Get margin from Fyerss

	// WebSocket endpoint
	mux.HandleFunc("/ws/events", WebSocketHandler(wsHub))

	// Live endpoints
	mux.HandleFunc("/live/ticks", LiveTicksHandler)
	mux.HandleFunc("/live/trades", LiveTradesHandler)

	// Backtest endpoints
	mux.HandleFunc("/backtest/run", BacktestRunHandler(dbPath))
	mux.HandleFunc("/backtest/trades", BacktestTradesHandler)
	mux.HandleFunc("/backtest/ticks", BacktestTicksHandler(dbPath))
	mux.HandleFunc("/backtest/data", BacktestDataHandler)

	return mux
}

func HFTStatusHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	executor.CurrentHFT.Status = "connected"
	status := map[string]interface{}{
		"hft": executor.CurrentHFT,
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "failed to encode status", http.StatusInternalServerError)
		return
	}
}

func DBQueryHandler(dbPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			http.Error(w, "query parameter is required", http.StatusBadRequest)
			return
		}

		// Initialize the database with the provided path
		db, err := sqlite.InitDefault(dbPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to initialize database: %v", err), http.StatusInternalServerError)
			return
		}

		result, err := db.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
