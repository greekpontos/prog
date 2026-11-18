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
	ErrPostNotFound = errors.New("post not found")
)

// PostRepo реализует PostRepository для работы с PostgreSQL
type PostRepo struct {
	db *sql.DB
}

// NewPostRepo создает новый экземпляр репозитория постов
func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

// Create сохраняет новый пост в БД и присваивает ему сгенерированный ID
func (r *PostRepo) Create(ctx context.Context, post *model.Post) error {
	query := `
		INSERT INTO posts (title, content, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	err := r.db.QueryRowContext(
		ctx, query,
		post.Title, post.Content, post.AuthorID, post.CreatedAt, post.UpdatedAt,
	).Scan(&post.ID)

	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// GetByID возвращает пост по его идентификатору
func (r *PostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		WHERE id = $1
	`

	var post model.Post
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID, &post.Title, &post.Content, &post.AuthorID, &post.CreatedAt, &post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return &post, nil
}

// GetAll возвращает список постов, отсортированных по дате создания и ID (новые сверху), с пагинацией
func (r *PostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}
	defer rows.Close()

	// Инициализируем пустой срез, чтобы в JSON возвращался [], а не null
	posts := make([]*model.Post, 0)
	for rows.Next() {
		var post model.Post
		if err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &post.AuthorID, &post.CreatedAt, &post.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate posts: %w", err)
	}

	return posts, nil
}

// GetTotalCount возвращает общее количество постов в таблице
func (r *PostRepo) GetTotalCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM posts`

	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return count, nil
}

// Update обновляет заголовок, текст и время изменения существующего поста с проверкой автора
func (r *PostRepo) Update(ctx context.Context, post *model.Post) error {
	post.UpdatedAt = time.Now()

	var (
		res sql.Result
		err error
	)

	if post.AuthorID > 0 {
		query := `
			UPDATE posts
			SET title = $1, content = $2, updated_at = $3
			WHERE id = $4 AND author_id = $5
		`
		res, err = r.db.ExecContext(ctx, query, post.Title, post.Content, post.UpdatedAt, post.ID, post.AuthorID)
	} else {
		query := `
			UPDATE posts
			SET title = $1, content = $2, updated_at = $3
			WHERE id = $4
		`
		res, err = r.db.ExecContext(ctx, query, post.Title, post.Content, post.UpdatedAt, post.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrPostNotFound
	}

	return nil
}

// Delete удаляет пост по ID
func (r *PostRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM posts WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrPostNotFound
	}

	return nil
}

// Exists проверяет, существует ли пост с заданным ID
func (r *PostRepo) Exists(ctx context.Context, id int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1)`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check post existence: %w", err)
	}

	return exists, nil
}

// GetByAuthorID возвращает посты конкретного автора с поддержкой пагинации и стабильной сортировкой
func (r *PostRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		WHERE author_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, authorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts by author: %w", err)
	}
	defer rows.Close()

	posts := make([]*model.Post, 0)
	for rows.Next() {
		var post model.Post
		if err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &post.AuthorID, &post.CreatedAt, &post.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate posts: %w", err)
	}

	return posts, nil
}

// GetCountByAuthorID возвращает общее количество постов конкретного автора
func (r *PostRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	query := `SELECT COUNT(*) FROM posts WHERE author_id = $1`

	var count int
	if err := r.db.QueryRowContext(ctx, query, authorID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count posts by author: %w", err)
	}

	return count, nil
}
