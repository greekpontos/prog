package handler

import (
	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/service"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// AuthHandler обрабатывает HTTP-запросы, связанные с регистрацией и аутентификацией
type AuthHandler struct {
	userService *service.UserService
}

// NewAuthHandler создает новый экземпляр обработчика аутентификации
func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

// Register обрабатывает POST /api/register для создания нового пользователя
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.UserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.userService.Register(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrUserAlreadyExists):
			writeError(w, "User with this email or username already exists", http.StatusConflict)
		default:
			writeError(w, "Failed to register user", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, resp, http.StatusCreated)
}

// Login обрабатывает POST /api/login для входа пользователя и выдачи JWT токена
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, "Invalid email or password", http.StatusUnauthorized)
		default:
			writeError(w, "Failed to login", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, resp, http.StatusOK)
}

// writeJSON сериализует данные в JSON и отправляет клиенту с указанным HTTP-статусом
func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError формирует и отправляет стандартизированный JSON-ответ об ошибке
func writeError(w http.ResponseWriter, message string, statusCode int) {
	writeJSON(w, model.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}, statusCode)
}

// getUserIDFromContext извлекает идентификатор авторизованного пользователя из контекста
func getUserIDFromContext(ctx context.Context) (int, bool) {
	return middleware.GetUserIDFromContext(ctx)
}
