package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// DB is a simple ORM-like facade exposing model stores.
type DB struct {
	Ticks  *TickStore
	Orders *OrderStore
	conn   *sql.DB
}

var (
	defaultOnce sync.Once
	defaultErr  error
	defaultPath string
	defaultDB   *DB
)

// InitDefault lazily opens a shared DB for the provided path (or fallback).
// Safe for concurrent use.
func InitDefault(dbPath string) (*DB, error) {
	defaultPath = dbPath
	defaultOnce.Do(func() {
		finalPath := defaultPath
		if finalPath == "" {
			finalPath = "hft.db"
		}
		store, err := NewDB(finalPath)
		if err != nil {
			defaultErr = err
			return
		}
		// // Seed a sample tick row only if empty.
		// if err := store.Ticks.SeedSample(context.Background()); err != nil {
		// 	defaultErr = err
		// 	return
		// }
		defaultDB = store
	})
	return defaultDB, defaultErr
}

// MustInitDefault initializes the shared DB or panics on failure.
func MustInitDefault(dbPath string) *DB {
	db, err := InitDefault(dbPath)
	if err != nil {
		panic(err)
	}
	return db
}

// DefaultDB returns the shared DB facade, initializing if needed.
func DefaultDB() *DB {
	db, _ := InitDefault(defaultPath)
	return db
}

// DefaultStore returns the shared TickStore for backwards compatibility.
func DefaultStore() *TickStore {
	db := DefaultDB()
	if db == nil {
		return nil
	}
	return db.Ticks
}

// NewDB opens (or creates) the SQLite database and initializes model stores.
func NewDB(dbPath string) (*DB, error) {
	if dbPath == "" {
		return nil, errors.New("db path is required")
	}
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}

	conn, err := sql.Open("sqlite", absPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	ticks := &TickStore{db: conn}
	if err := ticks.ensureSchema(); err != nil {
		return nil, err
	}

	orders := &OrderStore{db: conn}
	if err := orders.ensureSchema(); err != nil {
		return nil, err
	}

	return &DB{
		Ticks:  ticks,
		Orders: orders,
		conn:   conn,
	}, nil
}

// Close releases the underlying database connection.
func (d *DB) Close() error {
	if d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

// parseEpoch parses a date string to epoch seconds.
func parseEpoch(val string) (int64, bool) {
	if val == "" {
		return 0, false
	}
	if ts, err := strconv.ParseInt(val, 10, 64); err == nil {
		return ts, true
	}
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		return t.Unix(), true
	}
	if t, err := time.Parse("2006-01-02", val); err == nil {
		return t.Unix(), true
	}
	return 0, false
}

// nullID returns nil if id is 0, otherwise returns the id.
func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// Query executes a raw SQL query and returns the results as a slice of maps.
// Each map represents a row with column names as keys.
func (d *DB) Query(query string) ([]map[string]interface{}, error) {
	if d.conn == nil {
		return nil, errors.New("database connection is nil")
	}

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		// Convert the row to a map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for better JSON encoding
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return results, nil
}
