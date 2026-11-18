package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrEmptySecret  = errors.New("jwt secret key cannot be empty")
)

// Claims представляет полезную нагрузку (payload), хранимую внутри JWT токена
type Claims struct {
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTManager управляет генерацией и проверкой JWT токенов
type JWTManager struct {
	secretKey []byte
	ttl       time.Duration
}

// NewJWTManager создает новый экземпляр JWT менеджера с заданным секретом и временем жизни
func NewJWTManager(secretKey string, ttlHours int) *JWTManager {
	if ttlHours <= 0 {
		ttlHours = 24
	}
	return &JWTManager{
		secretKey: []byte(secretKey),
		ttl:       time.Duration(ttlHours) * time.Hour,
	}
}

// GenerateToken создает и подписывает новый JWT токен для пользователя
func (m *JWTManager) GenerateToken(userID int, email, username string) (string, time.Time, error) {
	if len(m.secretKey) == 0 {
		return "", time.Time{}, ErrEmptySecret
	}

	now := time.Now()
	expiresAt := now.Add(m.ttl)

	claims := &Claims{
		UserID:   userID,
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil
}

// ValidateToken проверяет подпись, допустимый алгоритм и срок действия JWT токена
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	if strings.TrimSpace(tokenString) == "" || len(m.secretKey) == 0 {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Явная проверка алгоритма подписи HMAC HS256
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidToken
			}
			return m.secretKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshToken перевыпускает новый JWT токен на основе действующего
func (m *JWTManager) RefreshToken(tokenString string) (string, time.Time, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", time.Time{}, err
	}
	return m.GenerateToken(claims.UserID, claims.Email, claims.Username)
}

// GetUserIDFromToken извлекает идентификатор пользователя из валидного токена
func (m *JWTManager) GetUserIDFromToken(tokenString string) (int, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}
