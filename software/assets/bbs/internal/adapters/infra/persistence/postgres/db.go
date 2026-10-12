package postgres

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sokoide/workshop/software/assets/bbs/internal/usecase"
)

//go:embed schema.sql
var schema string

// OpenDB connects to the exercise database and initializes its schema.
func OpenDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	fail := func(err error) (*sql.DB, error) { db.Close(); return nil, err }
	if err := db.PingContext(ctx); err != nil {
		return fail(fmt.Errorf("ping postgres: %w", err))
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fail(fmt.Errorf("begin migration: %w", err))
	}
	defer tx.Rollback()
	// Execute one statement at a time with pgx's default extended protocol.
	for _, statement := range strings.Split(schema, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fail(fmt.Errorf("migrate postgres: %w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return fail(fmt.Errorf("commit migration: %w", err))
	}
	return db, nil
}

var (
	_ usecase.BoardRepository    = (*BoardRepository)(nil)
	_ usecase.ThreadRepository   = (*ThreadRepository)(nil)
	_ usecase.PostRepository     = (*PostRepository)(nil)
	_ usecase.TransactionManager = (*TransactionManager)(nil)
)
