package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/domain/entity"
)

type ThreadRepository struct {
	db *sql.DB
}

func NewThreadRepository(db *sql.DB) *ThreadRepository {
	return &ThreadRepository{db: db}
}

func (r *ThreadRepository) FindByBoardID(ctx context.Context, boardID int64) ([]*entity.Thread, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, board_id, title, post_count, created_at, last_posted_at
		 FROM threads WHERE board_id = ? ORDER BY last_posted_at DESC`, boardID,
	)
	if err != nil {
		return nil, fmt.Errorf("query threads: %w", err)
	}
	defer rows.Close()

	var threads []*entity.Thread
	for rows.Next() {
		var m ThreadModel
		if err := rows.Scan(&m.ID, &m.BoardID, &m.Title, &m.PostCount, &m.CreatedAt, &m.LastPostedAt); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		threads = append(threads, m.ToEntity())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate threads: %w", err)
	}
	return threads, nil
}

func (r *ThreadRepository) FindByID(ctx context.Context, id int64) (*entity.Thread, error) {
	var m ThreadModel
	err := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, board_id, title, post_count, created_at, last_posted_at
		 FROM threads WHERE id = ?`, id,
	).Scan(&m.ID, &m.BoardID, &m.Title, &m.PostCount, &m.CreatedAt, &m.LastPostedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrThreadNotFound
		}
		return nil, fmt.Errorf("query thread by id: %w", err)
	}
	return m.ToEntity(), nil
}

func (r *ThreadRepository) Save(ctx context.Context, thread *entity.Thread) error {
	if thread.ID == 0 {
		res, err := executor(ctx, r.db).ExecContext(ctx,
			`INSERT INTO threads (board_id, title, post_count, created_at, last_posted_at)
			 VALUES (?, ?, ?, ?, ?)`,
			thread.BoardID, thread.Title, thread.PostCount, thread.CreatedAt, thread.LastPostedAt,
		)
		if err != nil {
			return fmt.Errorf("insert thread: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("get last insert id: %w", err)
		}
		thread.ID = id
		return nil
	}
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE threads SET post_count = ?, last_posted_at = ? WHERE id = ?`,
		thread.PostCount, thread.LastPostedAt, thread.ID,
	)
	if err != nil {
		return fmt.Errorf("update thread: %w", err)
	}
	return nil
}
