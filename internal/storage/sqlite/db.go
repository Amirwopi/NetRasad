package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
	log  *slog.Logger
	path string
}

func Open(ctx context.Context, path string, log *slog.Logger) (*DB, error) {
	if log == nil {
		log = slog.Default()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	q := url.Values{}
	q.Set("_pragma", "journal_mode(WAL)")
	q.Set("_pragma", "foreign_keys(ON)")
	q.Set("_pragma", "busy_timeout(5000)")
	q.Set("_pragma", "synchronous(NORMAL)")
	q.Set("_pragma", "temp_store(MEMORY)")

	dsn := "file:" + path + "?" + q.Encode()
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	conn.SetMaxOpenConns(4)
	conn.SetMaxIdleConns(2)

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	db := &DB{conn: conn, log: log, path: path}
	if err := db.migrate(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Info("sqlite database opened", "path", path)
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Path() string {
	return db.path
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}
