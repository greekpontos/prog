package repository

import (
	"blog-api/internal/model"
	"context"
)

// UserRepository определяет контракт для взаимодействия с хранилищем пользователей
type UserRepository interface {
	// Create создает нового пользователя в базе данных
	Create(ctx context.Context, user *model.User) error

	// GetByID получает пользователя по его уникальному ID
	GetByID(ctx context.Context, id int) (*model.User, error)

	// GetByEmail получает пользователя по адресу электронной почты
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// GetByUsername получает пользователя по имени пользователя
	GetByUsername(ctx context.Context, username string) (*model.User, error)

	// ExistsByEmail проверяет, занят ли указанный email
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByUsername проверяет, занят ли указанный username
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// PostRepository определяет контракт для взаимодействия с хранилищем постов
type PostRepository interface {
	// Create сохраняет новый пост в базе данных
	Create(ctx context.Context, post *model.Post) error

	// GetByID возвращает пост по его ID
	GetByID(ctx context.Context, id int) (*model.Post, error)

	// GetAll возвращает список постов с учетом пагинации (limit и offset)
	GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error)

	// GetTotalCount возвращает общее количество постов в базе
	GetTotalCount(ctx context.Context) (int, error)

	// GetByAuthorID возвращает посты конкретного автора с пагинацией
	GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error)

	// GetCountByAuthorID возвращает общее количество постов конкретного автора
	GetCountByAuthorID(ctx context.Context, authorID int) (int, error)

	// Update обновляет заголовок и содержимое существующего поста
	Update(ctx context.Context, post *model.Post) error

	// Delete удаляет пост по его ID
	Delete(ctx context.Context, id int) error

	// Exists проверяет, существует ли пост с указанным ID
	Exists(ctx context.Context, id int) (bool, error)
}

// CommentRepository определяет контракт для взаимодействия с хранилищем комментариев
type CommentRepository interface {
	// Create сохраняет новый комментарий к посту
	Create(ctx context.Context, comment *model.Comment) error

	// GetByID возвращает комментарий по его ID
	GetByID(ctx context.Context, id int) (*model.Comment, error)

	// GetByPostID возвращает комментарии к конкретному посту с пагинацией
	GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error)

	// GetCountByPostID возвращает количество комментариев к конкретному посту
	GetCountByPostID(ctx context.Context, postID int) (int, error)

	// GetByAuthorID возвращает комментарии конкретного автора с пагинацией
	GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Comment, error)

	// GetCountByAuthorID возвращает количество комментариев конкретного автора
	GetCountByAuthorID(ctx context.Context, authorID int) (int, error)

	// Update обновляет текст комментария
	Update(ctx context.Context, comment *model.Comment) error

	// Delete удаляет комментарий по его ID
	Delete(ctx context.Context, id int) error
}
