package middleware

import (
	"blog-api/internal/model"
	"blog-api/pkg/auth"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type contextKey string

const (
	UserIDKey    contextKey = "userID"
	UserEmailKey contextKey = "userEmail"
	UserNameKey  contextKey = "username"
)

// AuthMiddleware проверяет JWT токены и добавляет данные пользователя в контекст
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

// NewAuthMiddleware создает новый экземпляр middleware аутентификации
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// RequireAuth требует наличия валидного JWT токена в заголовке Authorization
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			writeJSONError(w, "Authorization header missing or invalid", http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtManager.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				writeJSONError(w, "Token has expired", http.StatusUnauthorized)
				return
			}
			writeJSONError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, UserNameKey, claims.Username)

		next(w, r.WithContext(ctx))
	}
}

// OptionalAuth извлекает данные пользователя из токена, если он передан
func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString != "" {
			if claims, err := m.jwtManager.ValidateToken(tokenString); err == nil {
				ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
				ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
				ctx = context.WithValue(ctx, UserNameKey, claims.Username)
				r = r.WithContext(ctx)
			}
		}
		next(w, r)
	}
}

// extractToken извлекает токен из заголовка формата "Bearer <token>"
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	token, found := strings.CutPrefix(authHeader, "Bearer ")
	if !found {
		token, found = strings.CutPrefix(authHeader, "bearer ")
	}
	if !found {
		return ""
	}
	return strings.TrimSpace(token)
}

// GetUserIDFromContext извлекает ID пользователя из контекста запроса
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}

// GetUserEmailFromContext извлекает email пользователя из контекста запроса
func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

// GetUsernameFromContext извлекает имя пользователя из контекста запроса
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UserNameKey).(string)
	return username, ok
}

// writeJSONError отправляет унифицированный JSON-ответ с ошибкой
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(model.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}
