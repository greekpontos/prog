package handler

import (
	"blog-api/internal/model"
	"blog-api/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const (
	defaultCommentLimit = 20
	maxCommentLimit     = 100
)

// CommentHandler обрабатывает HTTP-запросы для управления комментариями
type CommentHandler struct {
	commentService *service.CommentService
}

// NewCommentHandler создает новый экземпляр обработчика комментариев
func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// Create обрабатывает POST /api/posts/{postId}/comments (или POST /api/comments)
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Извлекаем postId из параметров маршрута Chi, если он передан в URL
	postIDStr := chi.URLParam(r, "postId")
	if postIDStr == "" {
		postIDStr = chi.URLParam(r, "id")
	}
	if postIDStr != "" {
		postID, err := strconv.Atoi(postIDStr)
		if err != nil || postID <= 0 {
			writeError(w, "Invalid post ID", http.StatusBadRequest)
			return
		}
		req.PostID = postID
	}

	comment, err := h.commentService.Create(r.Context(), userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrPostNotExists):
			writeError(w, "Post not found", http.StatusNotFound)
		default:
			writeError(w, "Failed to create comment", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, comment, http.StatusCreated)
}

// GetByID обрабатывает GET /api/comments/{id} для получения комментария по ID
func (h *CommentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeError(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCommentNotFound) {
			writeError(w, "Comment not found", http.StatusNotFound)
			return
		}
		writeError(w, "Failed to get comment", http.StatusInternalServerError)
		return
	}

	writeJSON(w, comment, http.StatusOK)
}

// GetByPost обрабатывает GET /api/posts/{postId}/comments для получения списка комментариев к посту
func (h *CommentHandler) GetByPost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postId")
	if postIDStr == "" {
		postIDStr = chi.URLParam(r, "id")
	}

	postID, err := strconv.Atoi(postIDStr)
	if err != nil || postID <= 0 {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	limit, offset, err := parsePaginationParams(r, defaultCommentLimit, maxCommentLimit)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	comments, total, err := h.commentService.GetByPost(r.Context(), postID, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrPostNotExists) {
			writeError(w, "Post not found", http.StatusNotFound)
			return
		}
		writeError(w, "Failed to get comments", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"comments": comments,
		"count":    len(comments),
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"post_id":  postID,
	}

	writeJSON(w, resp, http.StatusOK)
}

// Update обрабатывает PUT /api/comments/{id} для редактирования комментария автором
func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeError(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	var req model.CommentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.Update(r.Context(), id, userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrCommentNotFound):
			writeError(w, "Comment not found", http.StatusNotFound)
		case errors.Is(err, service.ErrForbidden):
			writeError(w, "You can only update your own comments", http.StatusForbidden)
		default:
			writeError(w, "Failed to update comment", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, comment, http.StatusOK)
}

// Delete обрабатывает DELETE /api/comments/{id} для удаления комментария автором
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeError(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	if err := h.commentService.Delete(r.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrCommentNotFound):
			writeError(w, "Comment not found", http.StatusNotFound)
		case errors.Is(err, service.ErrForbidden):
			writeError(w, "You can only delete your own comments", http.StatusForbidden)
		default:
			writeError(w, "Failed to delete comment", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
