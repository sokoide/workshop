package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sokoide/workshop/software/assets/bbs/internal/domain"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain/entity"
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
		 FROM posts WHERE thread_id = $1 ORDER BY number`, threadID,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []*entity.Post
	for rows.Next() {
		var m PostModel
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Number, &m.Author, &m.Body, &m.Sage, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
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
		`SELECT COUNT(*) FROM posts WHERE thread_id = $1`, threadID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count posts: %w", err)
	}
	return count, nil
}

func (r *PostRepository) Save(ctx context.Context, post *entity.Post) error {
	err := executor(ctx, r.db).QueryRowContext(ctx,
		`INSERT INTO posts (thread_id, number, author, body, sage, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		post.ThreadID, post.Number, post.Author, post.Body, post.Sage, post.CreatedAt,
	).Scan(&post.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "posts_thread_id_number_key" {
			return domain.ErrDuplicatePostNumber
		}
		return fmt.Errorf("insert post: %w", err)
	}
	return nil
}
