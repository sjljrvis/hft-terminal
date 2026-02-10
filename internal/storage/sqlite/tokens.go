package sqlite

import (
	"database/sql"
	"fmt"
)

// TokenStore provides helpers for storing broker auth tokens in SQLite.
type TokenStore struct {
	db *sql.DB
}

// ensureSchema creates the tokens table if it does not exist.
//
// Columns:
//   - access_token         TEXT, required
//   - refresh_token        TEXT, required
//   - expiry               INTEGER, required (Unix timestamp seconds or millis)
//   - refresh_token_expiry INTEGER, required (Unix timestamp seconds or millis)
//   - created_at           TEXT, default CURRENT_TIMESTAMP
//   - updated_at           TEXT, default CURRENT_TIMESTAMP
func (s *TokenStore) ensureSchema() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS tokens (
  id INTEGER PRIMARY KEY,
  access_token TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  expiry INTEGER NOT NULL,
  refresh_token_expiry INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tokens_access_token ON tokens(access_token);
`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("ensure tokens schema: %w", err)
	}

	// Migrate existing tables: add refresh_token_expiry column if it doesn't exist
	const migration = `
ALTER TABLE tokens ADD COLUMN refresh_token_expiry INTEGER DEFAULT 0;
`
	// This will fail silently if the column already exists, which is fine
	_, _ = s.db.Exec(migration)

	return nil
}

func (s *TokenStore) Create(accessToken, refreshToken string, expiry, refreshTokenExpiry int64) error {
	query := `
INSERT INTO tokens (access_token, refresh_token, expiry, refresh_token_expiry)
VALUES (?, ?, ?, ?)
`
	_, err := s.db.Exec(query, accessToken, refreshToken, expiry, refreshTokenExpiry)
	if err != nil {
		return fmt.Errorf("create token: %w", err)
	}
	return nil
}

func (s *TokenStore) Get() (string, string, int64, int64) {
	query := `
SELECT access_token, refresh_token, expiry, refresh_token_expiry
FROM tokens
WHERE id = 1
`
	row := s.db.QueryRow(query)
	var accessToken string
	var refreshToken string
	var expiry int64
	var refreshTokenExpiry int64
	err := row.Scan(&accessToken, &refreshToken, &expiry, &refreshTokenExpiry)
	if err != nil {
		return "", "", 0, 0
	}
	return accessToken, refreshToken, expiry, refreshTokenExpiry
}

func (s *TokenStore) Update(accessToken string, expiry int64) error {
	query := `
UPDATE tokens SET access_token = ?, expiry = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = 1
`
	_, err := s.db.Exec(query, accessToken, expiry)
	if err != nil {
		return fmt.Errorf("update token: %w", err)
	}
	return nil
}

func (s *TokenStore) UpdateWithRefreshToken(accessToken, refreshToken string, expiry, refreshTokenExpiry int64) error {
	query := `
UPDATE tokens SET access_token = ?, refresh_token = ?, expiry = ?, refresh_token_expiry = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = 1
`
	_, err := s.db.Exec(query, accessToken, refreshToken, expiry, refreshTokenExpiry)
	if err != nil {
		return fmt.Errorf("update token with refresh: %w", err)
	}
	return nil
}
