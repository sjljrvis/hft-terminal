package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"hft/internal/backtest"
	"hft/internal/storage/sqlite"
)

// BacktestRunHandler handles POST requests to run a backtest with custom date range.
func BacktestRunHandler(dbPath string) http.HandlerFunc {
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

		var request struct {
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if request.StartDate == "" || request.EndDate == "" {
			http.Error(w, "startDate and endDate are required", http.StatusBadRequest)
			return
		}

		// Run backtest with provided dates
		if err := backtest.RunWithDates(request.StartDate, request.EndDate); err != nil {
			http.Error(w, fmt.Sprintf("failed to run backtest: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":    "success",
			"startDate": request.StartDate,
			"endDate":   request.EndDate,
			"message":   "Backtest completed successfully",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// BacktestTradesHandler handles GET requests to return backtest trades dataframe.
func BacktestTradesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backtest.TradesToJSON()); err != nil {
		http.Error(w, "failed to encode trades", http.StatusInternalServerError)
		return
	}
}

// BacktestTicksHandler handles GET requests to return backtest ticks from the database.
func BacktestTicksHandler(dbPath string) http.HandlerFunc {
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

		// Get query parameters for filtering
		symbol := r.URL.Query().Get("symbol")
		tf := r.URL.Query().Get("tf")
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")

		// Initialize the database
		db, err := sqlite.InitDefault(dbPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to initialize database: %v", err), http.StatusInternalServerError)
			return
		}

		// Query ticks from database
		ctx := context.Background()
		ticks, err := db.Ticks.ListTicksFiltered(ctx, symbol, tf, 0, startDate, endDate)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to query ticks: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ticks); err != nil {
			http.Error(w, "failed to encode ticks", http.StatusInternalServerError)
			return
		}
	}
}

// BacktestDataHandler handles GET requests to return processed backtest data with indicators.
func BacktestDataHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(backtest.ToJSON()); err != nil {
		// Header already written, can't use http.Error, just log the error
		fmt.Printf("failed to encode backtest data: %v\n", err)
		return
	}
}
