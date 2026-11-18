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
	ErrCommentNotFound = errors.New("comment not found")
	ErrPostNotExists   = errors.New("post does not exist")
)

// CommentService реализует бизнес-логику управления комментариями к постам
type CommentService struct {
	commentRepo repository.CommentRepository
	postRepo    repository.PostRepository
	userRepo    repository.UserRepository
}

// NewCommentService создает новый экземпляр сервиса комментариев
func NewCommentService(
	commentRepo repository.CommentRepository,
	postRepo repository.PostRepository,
	userRepo repository.UserRepository,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
	}
}

// Create проверяет существование поста и создает новый комментарий
func (s *CommentService) Create(ctx context.Context, userID int, req *model.CommentCreateRequest) (*model.Comment, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if err := validateCommentCreateRequest(req); err != nil {
		return nil, err
	}

	// Убеждаемся, что комментируемый пост существует
	exists, err := s.postRepo.Exists(ctx, req.PostID)
	if err != nil {
		return nil, fmt.Errorf("failed to check post existence: %w", err)
	}
	if !exists {
		return nil, ErrPostNotExists
	}

	comment := &model.Comment{
		Content:  strings.TrimSpace(req.Content),
		PostID:   req.PostID,
		AuthorID: userID,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	if author, err := s.userRepo.GetByID(ctx, userID); err == nil {
		resp := author.ToResponse()
		comment.Author = &resp
	}

	return comment, nil
}

// GetByID возвращает комментарий по идентификатору
func (s *CommentService) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	if author, err := s.userRepo.GetByID(ctx, comment.AuthorID); err == nil {
		resp := author.ToResponse()
		comment.Author = &resp
	}

	return comment, nil
}

// GetByPost возвращает список комментариев к посту с пагинацией и общим количеством
func (s *CommentService) GetByPost(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Проверяем, что пост существует
	exists, err := s.postRepo.Exists(ctx, postID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to check post existence: %w", err)
	}
	if !exists {
		return nil, 0, ErrPostNotExists
	}

	comments, err := s.commentRepo.GetByPostID(ctx, postID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get comments: %w", err)
	}

	total, err := s.commentRepo.GetCountByPostID(ctx, postID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments: %w", err)
	}

	for _, c := range comments {
		if author, err := s.userRepo.GetByID(ctx, c.AuthorID); err == nil {
			resp := author.ToResponse()
			c.Author = &resp
		}
	}

	return comments, total, nil
}

// Update обновляет комментарий после проверки авторства
func (s *CommentService) Update(ctx context.Context, id int, userID int, req *model.CommentUpdateRequest) (*model.Comment, error) {
	if err := validateCommentUpdateRequest(req); err != nil {
		return nil, err
	}

	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	if !comment.CanBeEditedBy(userID) {
		return nil, ErrForbidden
	}

	comment.Content = strings.TrimSpace(req.Content)
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	if author, err := s.userRepo.GetByID(ctx, comment.AuthorID); err == nil {
		resp := author.ToResponse()
		comment.Author = &resp
	}

	return comment, nil
}

// Delete удаляет комментарий после проверки авторства
func (s *CommentService) Delete(ctx context.Context, id int, userID int) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return ErrCommentNotFound
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}

	if !comment.CanBeDeletedBy(userID) {
		return ErrForbidden
	}

	if err := s.commentRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return ErrCommentNotFound
		}
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return nil
}

// GetByAuthor возвращает комментарии конкретного автора с пагинацией
func (s *CommentService) GetByAuthor(ctx context.Context, authorID int, limit, offset int) ([]*model.Comment, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	comments, err := s.commentRepo.GetByAuthorID(ctx, authorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get comments by author: %w", err)
	}

	total, err := s.commentRepo.GetCountByAuthorID(ctx, authorID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count comments by author: %w", err)
	}

	return comments, total, nil
}

// validateCommentCreateRequest проверяет корректность данных при создании комментария
func validateCommentCreateRequest(req *model.CommentCreateRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request body is empty", ErrValidation)
	}
	if req.PostID <= 0 {
		return fmt.Errorf("%w: valid post_id is required", ErrValidation)
	}
	contentLen := len([]rune(strings.TrimSpace(req.Content)))
	if contentLen == 0 || contentLen > 1000 {
		return fmt.Errorf("%w: content must be between 1 and 1000 characters", ErrValidation)
	}
	return nil
}

// validateCommentUpdateRequest проверяет корректность данных при обновлении комментария
func validateCommentUpdateRequest(req *model.CommentUpdateRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request body is empty", ErrValidation)
	}
	contentLen := len([]rune(strings.TrimSpace(req.Content)))
	if contentLen == 0 || contentLen > 1000 {
		return fmt.Errorf("%w: content must be between 1 and 1000 characters", ErrValidation)
	}
	return nil
}
