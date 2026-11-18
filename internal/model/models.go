package model

import (
	"time"
)

// User представляет модель пользователя в системе
type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"` // Хешированный пароль, не отдаем в JSON
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Post представляет модель поста в блоге
type Post struct {
	ID        int           `json:"id" db:"id"`
	Title     string        `json:"title" db:"title"`
	Content   string        `json:"content" db:"content"`
	AuthorID  int           `json:"author_id" db:"author_id"`
	Author    *UserResponse `json:"author,omitempty" db:"-"` // Опциональные данные об авторе
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" db:"updated_at"`
}

// Comment представляет модель комментария к посту
type Comment struct {
	ID        int           `json:"id" db:"id"`
	Content   string        `json:"content" db:"content"`
	PostID    int           `json:"post_id" db:"post_id"`
	AuthorID  int           `json:"author_id" db:"author_id"`
	Author    *UserResponse `json:"author,omitempty" db:"-"` // Опциональные данные об авторе
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" db:"updated_at"`
}

// UserCreateRequest представляет входящий запрос на регистрацию пользователя
type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserLoginRequest представляет входящий запрос на вход пользователя
type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// PostCreateRequest представляет входящий запрос на создание поста
type PostCreateRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=200"`
	Content string `json:"content" validate:"required,min=1"`
}

// PostUpdateRequest представляет входящий запрос на обновление поста
type PostUpdateRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=200"`
	Content string `json:"content" validate:"required,min=1"`
}

// CommentCreateRequest представляет входящий запрос на создание комментария
type CommentCreateRequest struct {
	PostID  int    `json:"post_id"`
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// CommentUpdateRequest представляет входящий запрос на обновление комментария
type CommentUpdateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// UserResponse представляет публичные данные пользователя без конфиденциальных полей
type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenResponse представляет ответ при успешной аутентификации с JWT токеном
type TokenResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

// PostResponse представляет расширенный ответ с данными поста и его автором
type PostResponse struct {
	ID        int          `json:"id"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Author    UserResponse `json:"author"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// CommentResponse представляет расширенный ответ с данными комментария и его автором
type CommentResponse struct {
	ID        int          `json:"id"`
	Content   string       `json:"content"`
	PostID    int          `json:"post_id"`
	Author    UserResponse `json:"author"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ErrorResponse представляет унифицированный JSON-формат для возврата ошибок API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// ToResponse преобразует внутреннюю модель User в безопасную структуру UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

// CanBeEditedBy проверяет, является ли пользователь автором поста и может ли его редактировать
func (p *Post) CanBeEditedBy(userID int) bool {
	return p.AuthorID == userID
}

// CanBeDeletedBy проверяет, является ли пользователь автором поста и может ли его удалить
func (p *Post) CanBeDeletedBy(userID int) bool {
	return p.AuthorID == userID
}

// CanBeEditedBy проверяет, является ли пользователь автором комментария и может ли его редактировать
func (c *Comment) CanBeEditedBy(userID int) bool {
	return c.AuthorID == userID
}

// CanBeDeletedBy проверяет, является ли пользователь автором комментария и может ли его удалить
func (c *Comment) CanBeDeletedBy(userID int) bool {
	return c.AuthorID == userID
}
