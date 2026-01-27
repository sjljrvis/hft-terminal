package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"hft/pkg/types"
)

// OrderStore provides CRUD helpers for orders stored in SQLite.
type OrderStore struct {
	db *sql.DB
}

func (s *OrderStore) ensureSchema() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS orders (
  id INTEGER PRIMARY KEY,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  price REAL NOT NULL,
  quantity REAL NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_symbol ON orders(symbol);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("ensure orders schema: %w", err)
	}
	return nil
}

// FindAll returns all orders ordered by created_at asc.
func (s *OrderStore) FindAll(ctx context.Context) ([]types.Order, error) {
	query := `
SELECT id, symbol, side, price, quantity, status, created_at, updated_at
FROM orders
ORDER BY created_at ASC
`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []types.Order
	for rows.Next() {
		var o types.Order
		var created, updated string
		if err := rows.Scan(
			&o.ID,
			&o.Symbol,
			&o.Side,
			&o.Price,
			&o.Quantity,
			&o.Status,
			&created,
			&updated,
		); err != nil {
			return orders, fmt.Errorf("scan order: %w", err)
		}
		o.CreatedAt, _ = time.Parse(time.RFC3339, created)
		o.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return orders, fmt.Errorf("iterate orders: %w", err)
	}
	return orders, nil
}

// FindByID returns an order by its ID.
func (s *OrderStore) FindByID(ctx context.Context, id int64) (*types.Order, error) {
	query := `
SELECT id, symbol, side, price, quantity, status, created_at, updated_at
FROM orders
WHERE id = ?
`
	var o types.Order
	var created, updated string
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&o.ID,
		&o.Symbol,
		&o.Side,
		&o.Price,
		&o.Quantity,
		&o.Status,
		&created,
		&updated,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query order: %w", err)
	}
	o.CreatedAt, _ = time.Parse(time.RFC3339, created)
	o.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &o, nil
}

// Insert inserts a single order and returns the generated ID.
func (s *OrderStore) Insert(ctx context.Context, o types.Order) (int64, error) {
	created := o.CreatedAt
	updated := o.UpdatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	if updated.IsZero() {
		updated = created
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO orders (symbol, side, price, quantity, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, o.Symbol, o.Side, o.Price, o.Quantity, o.Status, created.Format(time.RFC3339), updated.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("insert order: %w", err)
	}
	return res.LastInsertId()
}

// Update updates an existing order by ID.
func (s *OrderStore) Update(ctx context.Context, o types.Order) error {
	updated := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
UPDATE orders
SET symbol = ?, side = ?, price = ?, quantity = ?, status = ?, updated_at = ?
WHERE id = ?
`, o.Symbol, o.Side, o.Price, o.Quantity, o.Status, updated.Format(time.RFC3339), o.ID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	return nil
}

// Delete removes an order by ID.
func (s *OrderStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	return nil
}
