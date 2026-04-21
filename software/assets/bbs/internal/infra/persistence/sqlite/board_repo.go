package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/domain/entity"
)

type BoardRepository struct {
	db *sql.DB
}

func NewBoardRepository(db *sql.DB) *BoardRepository {
	return &BoardRepository{db: db}
}

func (r *BoardRepository) FindAll(ctx context.Context) ([]*entity.Board, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `SELECT id, slug, name, created_at FROM boards ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query boards: %w", err)
	}
	defer rows.Close()

	var boards []*entity.Board
	for rows.Next() {
		var m BoardModel
		if err := rows.Scan(&m.ID, &m.Slug, &m.Name, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan board: %w", err)
		}
		boards = append(boards, m.ToEntity())
	}
	return boards, rows.Err()
}

func (r *BoardRepository) FindBySlug(ctx context.Context, slug string) (*entity.Board, error) {
	var m BoardModel
	err := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, slug, name, created_at FROM boards WHERE slug = ?`, slug,
	).Scan(&m.ID, &m.Slug, &m.Name, &m.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrBoardNotFound
		}
		return nil, fmt.Errorf("query board by slug: %w", err)
	}
	return m.ToEntity(), nil
}

func (r *BoardRepository) Save(ctx context.Context, board *entity.Board) error {
	if board.ID == 0 {
		res, err := executor(ctx, r.db).ExecContext(ctx,
			`INSERT INTO boards (slug, name, created_at) VALUES (?, ?, ?)`,
			board.Slug, board.Name, board.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert board: %w", err)
		}
		board.ID, _ = res.LastInsertId()
		return nil
	}
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE boards SET name = ? WHERE id = ?`,
		board.Name, board.ID,
	)
	if err != nil {
		return fmt.Errorf("update board: %w", err)
	}
	return nil
}
