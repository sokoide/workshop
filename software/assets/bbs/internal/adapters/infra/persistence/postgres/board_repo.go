package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sokoide/workshop/software/assets/bbs/internal/domain"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain/entity"
)

type BoardRepository struct {
	db *sql.DB
}

func NewBoardRepository(db *sql.DB) *BoardRepository {
	return &BoardRepository{db: db}
}

func (r *BoardRepository) FindAll(ctx context.Context) ([]*entity.Board, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx, `SELECT id, name, description, created_at FROM boards ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query boards: %w", err)
	}
	defer rows.Close()

	var boards []*entity.Board
	for rows.Next() {
		var m BoardModel
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan board: %w", err)
		}
		boards = append(boards, m.ToEntity())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate boards: %w", err)
	}
	return boards, nil
}

func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
	var m BoardModel
	err := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, name, description, created_at FROM boards WHERE name = $1`, name,
	).Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrBoardNotFound
		}
		return nil, fmt.Errorf("query board by name: %w", err)
	}
	return m.ToEntity(), nil
}

func (r *BoardRepository) Save(ctx context.Context, board *entity.Board) error {
	if board.ID == 0 {
		err := executor(ctx, r.db).QueryRowContext(ctx,
			`INSERT INTO boards (name, description, created_at) VALUES ($1, $2, $3) RETURNING id`,
			board.Name, board.Description, board.CreatedAt,
		).Scan(&board.ID)
		if err != nil {
			return fmt.Errorf("insert board: %w", err)
		}
		return nil
	}
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE boards SET description = $1 WHERE id = $2`,
		board.Description, board.ID,
	)
	if err != nil {
		return fmt.Errorf("update board: %w", err)
	}
	return nil
}
