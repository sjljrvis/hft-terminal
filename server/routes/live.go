package routes

import (
	"encoding/json"
	"net/http"

	"hft/internal/brokers"
	"hft/internal/executor"
)

func FyersMarginHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	brokers.GetMargin()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Margin fetched successfully"))
}

func FyersLoginHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// prinft query params
	authCode := r.URL.Query().Get("auth_code")
	brokers.Connect(authCode)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Login successful"))
}

// LiveTicksHandler handles GET requests to return live ticks.
func LiveTicksHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewEncoder(w).Encode(executor.ToJSON()); err != nil {
		http.Error(w, "failed to encode ticks", http.StatusInternalServerError)
		return
	}
}

// LiveTradesHandler handles GET requests to return live trades.
func LiveTradesHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewEncoder(w).Encode(executor.TradesToJSON()); err != nil {
		http.Error(w, "failed to encode trades", http.StatusInternalServerError)
		return
	}
}
