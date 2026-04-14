package authkeys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS client_key (
	hostname TEXT NOT NULL PRIMARY KEY,
	key_hash BLOB NOT NULL,
	created_at INTEGER NOT NULL
);
`

// Store holds per-client API key hashes in SQLite.
type Store struct {
	db *sql.DB
}

// DefaultPath returns the default auth database path under statsDir.
func DefaultPath(statsDir string) string {
	return filepath.Join(statsDir, "goprecords-auth.db")
}

// OpenStore opens or creates the SQLite auth database at path.
func OpenStore(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open auth db: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		db.Close()
		return nil, fmt.Errorf("pragma: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// EnsureSchema creates the client_key table if missing.
func (s *Store) EnsureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("auth schema: %w", err)
	}
	return nil
}

// KeyCount returns how many client keys are stored. When zero, upload auth is not enforced.
func (s *Store) KeyCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM client_key").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count keys: %w", err)
	}
	return n, nil
}

// CreateKey inserts or replaces the key for hostname and returns the plaintext token once.
func (s *Store) CreateKey(ctx context.Context, hostname string) (token string, err error) {
	if hostname == "" {
		return "", fmt.Errorf("empty hostname")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("random token: %w", err)
	}
	tok := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(tok))
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO client_key (hostname, key_hash, created_at) VALUES (?, ?, ?)
		 ON CONFLICT(hostname) DO UPDATE SET key_hash = excluded.key_hash, created_at = excluded.created_at`,
		hostname, sum[:], time.Now().Unix(),
	)
	if err != nil {
		return "", fmt.Errorf("insert key: %w", err)
	}
	return tok, nil
}

// Verify checks that token matches the stored hash for hostname.
func (s *Store) Verify(ctx context.Context, hostname, token string) (bool, error) {
	var stored []byte
	err := s.db.QueryRowContext(ctx, "SELECT key_hash FROM client_key WHERE hostname = ?", hostname).Scan(&stored)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lookup key: %w", err)
	}
	sum := sha256.Sum256([]byte(token))
	if len(stored) != len(sum) {
		return false, nil
	}
	return subtle.ConstantTimeCompare(stored, sum[:]) == 1, nil
}
