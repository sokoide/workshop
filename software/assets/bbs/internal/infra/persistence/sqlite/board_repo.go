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
		`SELECT id, name, description, created_at FROM boards WHERE name = ?`, name,
	).Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrBoardNotFound
		}
		return nil, fmt.Errorf("query board by name: %w", err)
	}
	return m.ToEntity(), nil
}

func (r *BoardRepository) Save(ctx context.Context, board *entity.Board) error {
	if board.ID == 0 {
		res, err := executor(ctx, r.db).ExecContext(ctx,
			`INSERT INTO boards (name, description, created_at) VALUES (?, ?, ?)`,
			board.Name, board.Description, board.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert board: %w", err)
		}
		board.ID, _ = res.LastInsertId()
		return nil
	}
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE boards SET description = ? WHERE id = ?`,
		board.Description, board.ID,
	)
	if err != nil {
		return fmt.Errorf("update board: %w", err)
	}
	return nil
}
