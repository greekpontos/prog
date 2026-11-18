package service

import (
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrPostNotFound = errors.New("post not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// PostService реализует бизнес-логику управления постами и проверку прав доступа
type PostService struct {
	postRepo repository.PostRepository
	userRepo repository.UserRepository
}

// NewPostService создает новый экземпляр сервиса постов
func NewPostService(postRepo repository.PostRepository, userRepo repository.UserRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

// Create валидирует данные и создает новый пост от имени текущего пользователя
func (s *PostService) Create(ctx context.Context, userID int, req *model.PostCreateRequest) (*model.Post, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if err := validatePostCreateRequest(req); err != nil {
		return nil, err
	}

	post := &model.Post{
		Title:    strings.TrimSpace(req.Title),
		Content:  strings.TrimSpace(req.Content),
		AuthorID: userID,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	// Добавляем публичную информацию об авторе в ответ
	if author, err := s.userRepo.GetByID(ctx, userID); err == nil {
		resp := author.ToResponse()
		post.Author = &resp
	}

	return post, nil
}

// GetByID возвращает пост по идентификатору и подгружает данные об авторе
func (s *PostService) GetByID(ctx context.Context, id int) (*model.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if author, err := s.userRepo.GetByID(ctx, post.AuthorID); err == nil {
		resp := author.ToResponse()
		post.Author = &resp
	}

	return post, nil
}

// GetAll возвращает список постов с нормализованной пагинацией и общим количеством записей
func (s *PostService) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	posts, err := s.postRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get posts: %w", err)
	}

	total, err := s.postRepo.GetTotalCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total posts count: %w", err)
	}

	for _, p := range posts {
		if author, err := s.userRepo.GetByID(ctx, p.AuthorID); err == nil {
			resp := author.ToResponse()
			p.Author = &resp
		}
	}

	return posts, total, nil
}

// Update обновляет пост после проверки, что текущий пользователь является его автором
func (s *PostService) Update(ctx context.Context, id int, userID int, req *model.PostUpdateRequest) (*model.Post, error) {
	if err := validatePostUpdateRequest(req); err != nil {
		return nil, err
	}

	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	// Проверяем права доступа: редактировать может только автор
	if !post.CanBeEditedBy(userID) {
		return nil, ErrForbidden
	}

	post.Title = strings.TrimSpace(req.Title)
	post.Content = strings.TrimSpace(req.Content)

	if err := s.postRepo.Update(ctx, post); err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	if author, err := s.userRepo.GetByID(ctx, post.AuthorID); err == nil {
		resp := author.ToResponse()
		post.Author = &resp
	}

	return post, nil
}

// Delete удаляет пост после проверки, что текущий пользователь является его автором
func (s *PostService) Delete(ctx context.Context, id int, userID int) error {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return ErrPostNotFound
		}
		return fmt.Errorf("failed to get post: %w", err)
	}

	// Проверяем права доступа: удалять может только автор
	if !post.CanBeDeletedBy(userID) {
		return ErrForbidden
	}

	if err := s.postRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return ErrPostNotFound
		}
		return fmt.Errorf("failed to delete post: %w", err)
	}

	return nil
}

// GetByAuthor возвращает посты указанного автора с пагинацией
func (s *PostService) GetByAuthor(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	posts, err := s.postRepo.GetByAuthorID(ctx, authorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get posts by author: %w", err)
	}

	total, err := s.postRepo.GetCountByAuthorID(ctx, authorID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts by author: %w", err)
	}

	for _, p := range posts {
		if author, err := s.userRepo.GetByID(ctx, p.AuthorID); err == nil {
			resp := author.ToResponse()
			p.Author = &resp
		}
	}

	return posts, total, nil
}

// validatePostCreateRequest проверяет корректность полей при создании поста
func validatePostCreateRequest(req *model.PostCreateRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request body is empty", ErrValidation)
	}
	titleLen := len([]rune(strings.TrimSpace(req.Title)))
	if titleLen == 0 || titleLen > 200 {
		return fmt.Errorf("%w: title must be between 1 and 200 characters", ErrValidation)
	}
	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("%w: content is required", ErrValidation)
	}
	return nil
}

// validatePostUpdateRequest проверяет корректность полей при обновлении поста
func validatePostUpdateRequest(req *model.PostUpdateRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request body is empty", ErrValidation)
	}
	titleLen := len([]rune(strings.TrimSpace(req.Title)))
	if titleLen == 0 || titleLen > 200 {
		return fmt.Errorf("%w: title must be between 1 and 200 characters", ErrValidation)
	}
	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("%w: content is required", ErrValidation)
	}
	return nil
}
