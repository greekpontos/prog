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
	defaultPostLimit = 10
	maxPostLimit     = 100
)

// PostHandler обрабатывает HTTP-запросы для управления постами блога
type PostHandler struct {
	postService *service.PostService
}

// NewPostHandler создает новый экземпляр обработчика постов
func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

// Create обрабатывает POST /api/posts для создания нового поста
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.PostCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	post, err := h.postService.Create(r.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeError(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	writeJSON(w, post, http.StatusCreated)
}

// GetByID обрабатывает GET /api/posts/{id} для получения поста по идентификатору
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.postService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			writeError(w, "Post not found", http.StatusNotFound)
			return
		}
		writeError(w, "Failed to get post", http.StatusInternalServerError)
		return
	}

	writeJSON(w, post, http.StatusOK)
}

// GetAll обрабатывает GET /api/posts для получения списка постов с пагинацией
func (h *PostHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePaginationParams(r, defaultPostLimit, maxPostLimit)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	posts, total, err := h.postService.GetAll(r.Context(), limit, offset)
	if err != nil {
		writeError(w, "Failed to get posts", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"posts":  posts,
		"count":  len(posts),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	writeJSON(w, resp, http.StatusOK)
}

// Update обрабатывает PUT /api/posts/{id} для редактирования поста автором
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var req model.PostUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	post, err := h.postService.Update(r.Context(), id, userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrPostNotFound):
			writeError(w, "Post not found", http.StatusNotFound)
		case errors.Is(err, service.ErrForbidden):
			writeError(w, "You can only update your own posts", http.StatusForbidden)
		default:
			writeError(w, "Failed to update post", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, post, http.StatusOK)
}

// Delete обрабатывает DELETE /api/posts/{id} для удаления поста автором
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	if err := h.postService.Delete(r.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrPostNotFound):
			writeError(w, "Post not found", http.StatusNotFound)
		case errors.Is(err, service.ErrForbidden):
			writeError(w, "You can only delete your own posts", http.StatusForbidden)
		default:
			writeError(w, "Failed to delete post", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetByAuthor обрабатывает GET /api/posts/author/{authorID} для получения постов автора
func (h *PostHandler) GetByAuthor(w http.ResponseWriter, r *http.Request) {
	authorID, err := strconv.Atoi(chi.URLParam(r, "authorID"))
	if err != nil || authorID <= 0 {
		writeError(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	limit, offset, err := parsePaginationParams(r, defaultPostLimit, maxPostLimit)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	posts, total, err := h.postService.GetByAuthor(r.Context(), authorID, limit, offset)
	if err != nil {
		writeError(w, "Failed to get author posts", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"posts":  posts,
		"count":  len(posts),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	writeJSON(w, resp, http.StatusOK)
}

// parsePaginationParams разбирает и нормализует параметры пагинации limit и offset
func parsePaginationParams(r *http.Request, defaultLimit, maxLimit int) (int, int, error) {
	query := r.URL.Query()
	limit := defaultLimit
	offset := 0

	if limitStr := query.Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, errors.New("invalid limit parameter")
		}
		if parsedLimit <= 0 {
			limit = defaultLimit
		} else if parsedLimit > maxLimit {
			limit = maxLimit
		} else {
			limit = parsedLimit
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, errors.New("invalid offset parameter")
		}
		if parsedOffset < 0 {
			offset = 0
		} else {
			offset = parsedOffset
		}
	}

	return limit, offset, nil
}
