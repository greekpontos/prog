package repository

import (
	"blog-api/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
)

// CommentRepo реализует CommentRepository для работы с PostgreSQL
type CommentRepo struct {
	db *sql.DB
}

// NewCommentRepo создает новый экземпляр репозитория комментариев
func NewCommentRepo(db *sql.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

// Create добавляет новый комментарий к посту и возвращает его ID
func (r *CommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	query := `
		INSERT INTO comments (content, post_id, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	err := r.db.QueryRowContext(
		ctx, query,
		comment.Content, comment.PostID, comment.AuthorID, comment.CreatedAt, comment.UpdatedAt,
	).Scan(&comment.ID)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// GetByID возвращает комментарий по его ID
func (r *CommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	query := `
		SELECT id, content, post_id, author_id, created_at, updated_at
		FROM comments
		WHERE id = $1
	`

	var comment model.Comment
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&comment.ID, &comment.Content, &comment.PostID,
		&comment.AuthorID, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return &comment, nil
}

// GetByPostID возвращает список комментариев к посту (в хронологическом порядке) с пагинацией и вторичной сортировкой по ID
func (r *CommentRepo) GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	query := `
		SELECT id, content, post_id, author_id, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	comments := make([]*model.Comment, 0)
	for rows.Next() {
		var comment model.Comment
		if err := rows.Scan(
			&comment.ID, &comment.Content, &comment.PostID,
			&comment.AuthorID, &comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate comments: %w", err)
	}

	return comments, nil
}

// GetCountByPostID возвращает общее количество комментариев к указанному посту
func (r *CommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE post_id = $1`

	var count int
	if err := r.db.QueryRowContext(ctx, query, postID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}

	return count, nil
}

// GetByAuthorID возвращает комментарии конкретного автора с пагинацией и стабильной сортировкой
func (r *CommentRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Comment, error) {
	query := `
		SELECT id, content, post_id, author_id, created_at, updated_at
		FROM comments
		WHERE author_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, authorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments by author: %w", err)
	}
	defer rows.Close()

	comments := make([]*model.Comment, 0)
	for rows.Next() {
		var comment model.Comment
		if err := rows.Scan(
			&comment.ID, &comment.Content, &comment.PostID,
			&comment.AuthorID, &comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate comments: %w", err)
	}

	return comments, nil
}

// GetCountByAuthorID возвращает общее количество комментариев автора
func (r *CommentRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE author_id = $1`

	var count int
	if err := r.db.QueryRowContext(ctx, query, authorID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count comments by author: %w", err)
	}

	return count, nil
}

// Update обновляет содержимое комментария с проверкой автора
func (r *CommentRepo) Update(ctx context.Context, comment *model.Comment) error {
	comment.UpdatedAt = time.Now()

	var (
		res sql.Result
		err error
	)

	if comment.AuthorID > 0 {
		query := `
			UPDATE comments
			SET content = $1, updated_at = $2
			WHERE id = $3 AND author_id = $4
		`
		res, err = r.db.ExecContext(ctx, query, comment.Content, comment.UpdatedAt, comment.ID, comment.AuthorID)
	} else {
		query := `
			UPDATE comments
			SET content = $1, updated_at = $2
			WHERE id = $3
		`
		res, err = r.db.ExecContext(ctx, query, comment.Content, comment.UpdatedAt, comment.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrCommentNotFound
	}

	return nil
}

// Delete удаляет комментарий по ID
func (r *CommentRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM comments WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrCommentNotFound
	}

	return nil
}
