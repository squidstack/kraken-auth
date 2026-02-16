package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	// pgx stdlib driver for database/sql
	_ "github.com/jackc/pgx/v5/stdlib"

	"kraken-auth/internal/logger"
)

// Init is a small convenience that opens using env vars.
// It waits briefly for the DB to come up and pings it.
func Init() (*sql.DB, error) {
	return OpenFromEnv(context.Background())
}

// OpenFromEnv opens a *sql.DB using AUTH_DB_URL / AUTH_DB_USER / AUTH_DB_PASSWORD.
// It converts jdbc URLs to postgres URLs, injects user/pass if provided,
// sets sane pool options, and waits up to ~20s for a successful ping.
func OpenFromEnv(ctx context.Context) (*sql.DB, error) {
	raw := strings.TrimSpace(os.Getenv("AUTH_DB_URL"))
	user := os.Getenv("AUTH_DB_USER")
	pass := os.Getenv("AUTH_DB_PASSWORD")

	if raw == "" {
		return nil, fmt.Errorf("AUTH_DB_URL is empty")
	}

	dsn, err := normalizeDSN(raw, user, pass)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open(pgx): %w", err)
	}

	// Connection pool knobs (tweak as needed)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Wait for DB to accept connections.
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error
	attempt := 0
	for time.Now().Before(deadline) {
		attempt++
		ctxPing, cancel := context.WithTimeout(ctx, 2*time.Second)
		lastErr = db.PingContext(ctxPing)
		cancel()
		if lastErr == nil {
			logger.Infof("[db] connected to %s", redactedDSN(dsn))
			return db, nil
		}
		if attempt <= 2 {
			logger.Debugf("[db] ping attempt=%d failed: %v", attempt, lastErr)
		} else if attempt%5 == 0 {
			logger.Warnf("[db] still waiting for DB (attempt=%d): %v", attempt, lastErr)
		}
		time.Sleep(1 * time.Second)
	}

	_ = db.Close()
	return nil, fmt.Errorf("db.Ping: %w", lastErr)
}

// normalizeDSN accepts either jdbc:postgresql://... or postgres://...
// It ensures a postgres URL is returned and injects username/password
// (from env) if provided. If no sslmode is present, defaults to disable.
func normalizeDSN(raw, username, password string) (string, error) {
	uStr := strings.TrimSpace(raw)

	// Strip "jdbc:" prefix if present.
	if strings.HasPrefix(strings.ToLower(uStr), "jdbc:") {
		uStr = uStr[len("jdbc:"):]
	}

	// If scheme missing, assume postgres
	if !strings.Contains(uStr, "://") {
		uStr = "postgres://" + uStr
	}

	u, err := url.Parse(uStr)
	if err != nil {
		return "", fmt.Errorf("cannot parse `%s`: %w", raw, err)
	}

	// Inject user/pass if given (env overrides URL creds)
	if username != "" {
		if password != "" {
			u.User = url.UserPassword(username, password)
		} else {
			u.User = url.User(username)
		}
	}

	// Default sslmode=disable if not specified
	q := u.Query()
	if q.Get("sslmode") == "" {
		q.Set("sslmode", "disable")
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// redactedDSN hides the password in logs.
func redactedDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User != nil {
		name := u.User.Username()
		if _, has := u.User.Password(); has {
			u.User = url.UserPassword(name, "****")
		}
	}
	return u.String()
}
