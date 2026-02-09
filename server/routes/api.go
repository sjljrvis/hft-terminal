package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"hft/internal/backtest"
	"hft/internal/execution"
	"hft/internal/storage/sqlite"
	sqliteStore "hft/internal/storage/sqlite"
)

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// APIHandler exposes the API HTTP handler.
func APIHandler(mode string, dbPath string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hft/status", HFTStatusHandler)
	mux.HandleFunc("/trades", TradesHandler)
	mux.HandleFunc("/ticks", TicksHandler(dbPath))
	mux.HandleFunc("/db/query", DBQueryHandler(dbPath))
	return mux
}

func HFTStatusHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	execution.CurrentHFT.Status = "connected"
	status := map[string]interface{}{
		"hft": execution.CurrentHFT,
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "failed to encode status", http.StatusInternalServerError)
		return
	}
}

// TicksHandler returns ticks from the SQLite store.
func TicksHandler(dbPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		// return backtest.Current.DF as json
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(backtest.ToJSON())
	}
}

func TradesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backtest.TradesToJSON())
}

var (
	tickStoreOnce sync.Once
	tickStore     *sqliteStore.TickStore
	tickStoreErr  error
	tickDBPath    string
)

func getTickStore(ctx context.Context, dbPath string) (*sqliteStore.TickStore, error) {
	initCtx := context.Background()
	tickDBPath = dbPath
	tickStoreOnce.Do(func() {
		finalPath := tickDBPath
		if finalPath == "" {
			finalPath = "hft.db"
		}
		store, err := sqliteStore.NewTickStore(finalPath)
		if err != nil {
			tickStoreErr = err
			log.Printf("tick store init failed (%s): %v", finalPath, err)
			return
		}
		if err := store.SeedSample(initCtx); err != nil {
			tickStoreErr = err
			log.Printf("tick store seed failed (%s): %v", finalPath, err)
			return
		}
		tickStore = store
	})
	return tickStore, tickStoreErr
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
