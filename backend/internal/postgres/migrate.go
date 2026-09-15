package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Edward-McCain/webhooks_Go/backend/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Migrate applies all pending database migrations.
func Migrate(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// MigrateWithContext respects cancellation before opening the database.
func MigrateWithContext(ctx context.Context, databaseURL string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return Migrate(databaseURL)
	}
}
