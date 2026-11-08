package store

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

var dbName = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// splitDSN returns the database name of dsn and a DSN of the same server's
// postgres maintenance database. No psql or Docker is needed on the dev Mac.
func splitDSN(dsn string) (name, admin string, err error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	name = strings.TrimPrefix(u.Path, "/")
	if !dbName.MatchString(name) {
		return "", "", fmt.Errorf("database name %q is not a plain identifier", name)
	}
	u.Path = "/postgres"
	return name, u.String(), nil
}

// EnsureDatabase creates the database named in dsn when it does not exist.
func EnsureDatabase(ctx context.Context, dsn string) (created bool, err error) {
	name, admin, err := splitDSN(dsn)
	if err != nil {
		return false, err
	}
	c, err := pgx.Connect(ctx, admin)
	if err != nil {
		return false, fmt.Errorf("connect admin: %w", err)
	}
	defer c.Close(ctx)
	var exists bool
	if err := c.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return false, fmt.Errorf("check database: %w", err)
	}
	if exists {
		return false, nil
	}
	if _, err := c.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		return false, fmt.Errorf("create database %s: %w", name, err)
	}
	return true, nil
}

// DropMockDatabase drops the database named in dsn; it refuses any name that
// does not end in _mock, so the real database is never touched.
func DropMockDatabase(ctx context.Context, dsn string) error {
	name, admin, err := splitDSN(dsn)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(name, "_mock") {
		return fmt.Errorf("refusing to drop %q: only *_mock databases", name)
	}
	c, err := pgx.Connect(ctx, admin)
	if err != nil {
		return fmt.Errorf("connect admin: %w", err)
	}
	defer c.Close(ctx)
	if _, err := c.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
		return fmt.Errorf("drop database %s: %w", name, err)
	}
	return nil
}
