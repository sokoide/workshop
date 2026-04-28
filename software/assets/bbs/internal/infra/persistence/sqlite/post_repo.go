package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/domain/entity"
)

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) FindByThreadID(ctx context.Context, threadID int64) ([]*entity.Post, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, thread_id, number, author, body, sage, created_at
		 FROM posts WHERE thread_id = ? ORDER BY number`, threadID,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []*entity.Post
	for rows.Next() {
		var m PostModel
		var sage int
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Number, &m.Author, &m.Body, &sage, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		m.Sage = sage == 1
		posts = append(posts, m.ToEntity())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	return posts, nil
}

func (r *PostRepository) CountByThreadID(ctx context.Context, threadID int64) (int, error) {
	var count int
	err := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM posts WHERE thread_id = ?`, threadID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count posts: %w", err)
	}
	return count, nil
}

func (r *PostRepository) Save(ctx context.Context, post *entity.Post) error {
	sage := 0
	if post.Sage {
		sage = 1
	}
	res, err := executor(ctx, r.db).ExecContext(ctx,
		`INSERT INTO posts (thread_id, number, author, body, sage, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		post.ThreadID, post.Number, post.Author, post.Body, sage, post.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.ErrDuplicatePostNumber
		}
		return fmt.Errorf("insert post: %w", err)
	}
	post.ID, _ = res.LastInsertId()
	return nil
}
