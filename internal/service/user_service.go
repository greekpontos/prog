package service

import (
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/pkg/auth"
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrValidation         = errors.New("validation error")
)

// UserService реализует бизнес-логику управления пользователями и аутентификации
type UserService struct {
	userRepo   repository.UserRepository
	jwtManager *auth.JWTManager
}

// NewUserService создает новый экземпляр сервиса пользователей
func NewUserService(userRepo repository.UserRepository, jwtManager *auth.JWTManager) *UserService {
	return &UserService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// Register валидирует данные, регистрирует нового пользователя и возвращает JWT токен
func (s *UserService) Register(ctx context.Context, req *model.UserCreateRequest) (*model.TokenResponse, error) {
	if err := validateUserCreateRequest(req); err != nil {
		return nil, err
	}

	// Нормализуем email и username
	email := strings.TrimSpace(strings.ToLower(req.Email))
	username := strings.TrimSpace(req.Username)

	// Проверяем уникальность email
	emailExists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if emailExists {
		return nil, ErrUserAlreadyExists
	}

	// Проверяем уникальность username
	usernameExists, err := s.userRepo.ExistsByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
	}
	if usernameExists {
		return nil, ErrUserAlreadyExists
	}

	// Хешируем пароль с помощью bcrypt
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	user := &model.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
	}

	// Сохраняем пользователя в базе данных
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Генерируем JWT токен для созданного пользователя
	token, expiresAt, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	}, nil
}

// Login проверяет учетные данные пользователя и выдает JWT токен
func (s *UserService) Login(ctx context.Context, req *model.UserLoginRequest) (*model.TokenResponse, error) {
	if err := validateUserLoginRequest(req); err != nil {
		return nil, err
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))

	// Ищем пользователя по email (при отсутствии возвращаем общую ошибку ErrInvalidCredentials)
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Сверяем введенный пароль с хешем в базе
	if !auth.CheckPassword(req.Password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	// Генерируем JWT токен
	token, expiresAt, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	}, nil
}

// ValidateToken проверяет валидность JWT токена и возвращает claims пользователя
func (s *UserService) ValidateToken(tokenString string) (*auth.Claims, error) {
	return s.jwtManager.ValidateToken(tokenString)
}

// GetByID возвращает пользователя по его идентификатору
func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail возвращает пользователя по адресу электронной почты
func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// validateUserCreateRequest проверяет корректность полей при регистрации
func validateUserCreateRequest(req *model.UserCreateRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request body is empty", ErrValidation)
	}
	usernameLen := len([]rune(strings.TrimSpace(req.Username)))
	if usernameLen < 3 || usernameLen > 50 {
		return fmt.Errorf("%w: username must be between 3 and 50 characters", ErrValidation)
	}
	email := strings.TrimSpace(req.Email)
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: invalid email address", ErrValidation)
	}
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return nil
}

// validateUserLoginRequest проверяет корректность полей при входе
func validateUserLoginRequest(req *model.UserLoginRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request body is empty", ErrValidation)
	}
	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		return fmt.Errorf("%w: email and password are required", ErrValidation)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: invalid email address", ErrValidation)
	}
	return nil
}
