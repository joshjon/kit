package sqlitedb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	_ "modernc.org/sqlite"
)

const (
	healthRetryInterval = time.Second
	healthMaxRetries    = 5
)

type OpenOption func(opts *openOpts)

// WithDir sets the directory used to store the SQLite database file.
func WithDir(dir string) OpenOption {
	return func(opts *openOpts) {
		opts.dir = dir
	}
}

// WithDBName sets the SQLite database name used when creating the `<dbName>.db`
// file. This option has no effect when WithInMemory is used.
func WithDBName(dbName string) OpenOption {
	return func(opts *openOpts) {
		opts.dbName = dbName
	}
}

// WithInMemory configures the connection to use an in-memory SQLite database.
func WithInMemory() OpenOption {
	return func(opts *openOpts) {
		opts.inMemory = true
	}
}

type openOpts struct {
	dir      string
	dbName   string
	inMemory bool
}

func Open(ctx context.Context, opts ...OpenOption) (*sql.DB, error) {
	var o openOpts
	for _, opt := range opts {
		opt(&o)
	}

	var dsn string

	if o.inMemory {
		dsn = ":memory:"
	} else {
		if o.dbName == "" {
			o.dbName = "app"
		}
		file := o.dbName + ".db"
		if o.dir != "" {
			if err := os.MkdirAll(o.dir, 0755); err != nil {
				return nil, fmt.Errorf("create sqlite directory: %w", err)
			}
			file = strings.TrimSuffix(o.dir, "/") + "/" + file
		}
		dsn = "file:" + file + "?_journal_mode=WAL&_busy_timeout=5000"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Set max connections to 1 since sqlite only supports a single writer at a time
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys = on;"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if !o.inMemory {
		// Use WAL mode only for file backed DBs
		if _, err = db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
			return nil, fmt.Errorf("enable WAL mode: %w", err)
		}
	}

	if err = waitHealthy(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

func waitHealthy(ctx context.Context, db *sql.DB) error {
	pingFn := func() error {
		pctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return db.PingContext(pctx)
	}
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(healthRetryInterval), healthMaxRetries)
	if err := backoff.Retry(pingFn, bo); err != nil {
		return fmt.Errorf("sqlite connection unhealthy: %w", err)
	}
	return nil
}
