package service

import (
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/pkg/auth"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// --- Моки репозиториев для тестирования сервисного слоя ---

type mockUserRepo struct {
	users  map[int]*model.User
	nextID int
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:  make(map[int]*model.User),
		nextID: 1,
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	for _, u := range m.users {
		if u.Email == user.Email || u.Username == user.Username {
			return repository.ErrUserExists
		}
	}
	user.ID = m.nextID
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	m.nextID++
	cp := *user
	m.users[user.ID] = &cp
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	if u, ok := m.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			cp := *u
			return &cp, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	for _, u := range m.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	for _, u := range m.users {
		if u.Username == username {
			return true, nil
		}
	}
	return false, nil
}

type mockPostRepo struct {
	posts  map[int]*model.Post
	nextID int
}

func newMockPostRepo() *mockPostRepo {
	return &mockPostRepo{
		posts:  make(map[int]*model.Post),
		nextID: 1,
	}
}

func (m *mockPostRepo) Create(ctx context.Context, post *model.Post) error {
	post.ID = m.nextID
	post.CreatedAt = time.Now()
	post.UpdatedAt = post.CreatedAt
	m.nextID++
	cp := *post
	m.posts[post.ID] = &cp
	return nil
}

func (m *mockPostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	if p, ok := m.posts[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, repository.ErrPostNotFound
}

func (m *mockPostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	var all []*model.Post
	for i := 1; i < m.nextID; i++ {
		if p, ok := m.posts[i]; ok {
			cp := *p
			all = append(all, &cp)
		}
	}
	if offset >= len(all) {
		return []*model.Post{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *mockPostRepo) GetTotalCount(ctx context.Context) (int, error) {
	return len(m.posts), nil
}

func (m *mockPostRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error) {
	var res []*model.Post
	for _, p := range m.posts {
		if p.AuthorID == authorID {
			cp := *p
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (m *mockPostRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	count := 0
	for _, p := range m.posts {
		if p.AuthorID == authorID {
			count++
		}
	}
	return count, nil
}

func (m *mockPostRepo) Update(ctx context.Context, post *model.Post) error {
	existing, ok := m.posts[post.ID]
	if !ok {
		return repository.ErrPostNotFound
	}
	if post.AuthorID > 0 && existing.AuthorID != post.AuthorID {
		return repository.ErrPostNotFound
	}
	post.UpdatedAt = time.Now()
	cp := *post
	m.posts[post.ID] = &cp
	return nil
}

func (m *mockPostRepo) Delete(ctx context.Context, id int) error {
	if _, ok := m.posts[id]; !ok {
		return repository.ErrPostNotFound
	}
	delete(m.posts, id)
	return nil
}

func (m *mockPostRepo) Exists(ctx context.Context, id int) (bool, error) {
	_, ok := m.posts[id]
	return ok, nil
}

type mockCommentRepo struct {
	comments map[int]*model.Comment
	nextID   int
}

func newMockCommentRepo() *mockCommentRepo {
	return &mockCommentRepo{
		comments: make(map[int]*model.Comment),
		nextID:   1,
	}
}

func (m *mockCommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	comment.ID = m.nextID
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = comment.CreatedAt
	m.nextID++
	cp := *comment
	m.comments[comment.ID] = &cp
	return nil
}

func (m *mockCommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	if c, ok := m.comments[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, repository.ErrCommentNotFound
}

func (m *mockCommentRepo) GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	var res []*model.Comment
	for _, c := range m.comments {
		if c.PostID == postID {
			cp := *c
			res = append(res, &cp)
		}
	}
	return res, nil
}

func (m *mockCommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	count := 0
	for _, c := range m.comments {
		if c.PostID == postID {
			count++
		}
	}
	return count, nil
}

func (m *mockCommentRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Comment, error) {
	return []*model.Comment{}, nil
}

func (m *mockCommentRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	return 0, nil
}

func (m *mockCommentRepo) Update(ctx context.Context, comment *model.Comment) error {
	if _, ok := m.comments[comment.ID]; !ok {
		return repository.ErrCommentNotFound
	}
	cp := *comment
	m.comments[comment.ID] = &cp
	return nil
}

func (m *mockCommentRepo) Delete(ctx context.Context, id int) error {
	if _, ok := m.comments[id]; !ok {
		return repository.ErrCommentNotFound
	}
	delete(m.comments, id)
	return nil
}

// --- Тесты сценариев бизнес-логики ---

func TestUserService_RegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	jwtMgr := auth.NewJWTManager("test-secret-key-with-sufficient-length-32chars", 24)
	userSvc := NewUserService(userRepo, jwtMgr)

	// 1. Регистрация с некорректными данными (пустое имя, пробелы, неверный email, слабый пароль)
	invalidRequests := []*model.UserCreateRequest{
		{Username: "   ", Email: "user@example.com", Password: "password123"},
		{Username: "alice", Email: "invalid-email", Password: "password123"},
		{Username: "alice", Email: "alice@example.com", Password: "123"},
	}
	for _, req := range invalidRequests {
		if _, err := userSvc.Register(ctx, req); !errors.Is(err, ErrValidation) {
			t.Fatalf("expected ErrValidation for %+v, got %v", req, err)
		}
	}

	// 2. Успешная регистрация
	resp, err := userSvc.Register(ctx, &model.UserCreateRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "secretPassword123",
	})
	if err != nil {
		t.Fatalf("unexpected error on register: %v", err)
	}
	if resp.Token == "" || resp.User.Email != "alice@example.com" {
		t.Fatalf("unexpected register response: %+v", resp)
	}

	// 3. Повторная регистрация с существующим email
	_, err = userSvc.Register(ctx, &model.UserCreateRequest{
		Username: "alice_other",
		Email:    "alice@example.com",
		Password: "secretPassword123",
	})
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}

	// 4. Вход с неправильным паролем
	_, err = userSvc.Login(ctx, &model.UserLoginRequest{
		Email:    "alice@example.com",
		Password: "wrongPassword",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	// 5. Успешный вход
	loginResp, err := userSvc.Login(ctx, &model.UserLoginRequest{
		Email:    "alice@example.com",
		Password: "secretPassword123",
	})
	if err != nil || loginResp.Token == "" {
		t.Fatalf("expected successful login, got err=%v", err)
	}
}

func TestJWT_Expiration(t *testing.T) {
	secret := "test-secret-key-with-sufficient-length-32chars"
	jwtMgr := auth.NewJWTManager(secret, 1)

	// Генерируем токен с истекшим сроком действия
	expiredClaims := &auth.Claims{
		UserID:   1,
		Email:    "user@example.com",
		Username: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	signedExpired, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	if _, err := jwtMgr.ValidateToken(signedExpired); !errors.Is(err, auth.ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestPostService_PermissionsAndPagination(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	postRepo := newMockPostRepo()
	postSvc := NewPostService(postRepo, userRepo)

	_ = userRepo.Create(ctx, &model.User{Username: "author1", Email: "a1@example.com", Password: "hashedpassword"})
	_ = userRepo.Create(ctx, &model.User{Username: "author2", Email: "a2@example.com", Password: "hashedpassword"})

	// Создаем пост от лица пользователя 1
	post, err := postSvc.Create(ctx, 1, &model.PostCreateRequest{
		Title:   "First Post",
		Content: "Valid content",
	})
	if err != nil {
		t.Fatalf("unexpected error creating post: %v", err)
	}

	// Попытка изменения чужой публикации пользователем 2 -> ErrForbidden
	_, err = postSvc.Update(ctx, post.ID, 2, &model.PostUpdateRequest{
		Title:   "Hacked Title",
		Content: "Hacked Content",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when editing another user's post, got %v", err)
	}

	// Попытка удаления чужой публикации пользователем 2 -> ErrForbidden
	if err := postSvc.Delete(ctx, post.ID, 2); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when deleting another user's post, got %v", err)
	}
	// Обращение к несуществующему посту -> ErrPostNotFound
	if _, err := postSvc.GetByID(ctx, 9999); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}

	// Проверка граничных параметров пагинации (отрицательный offset, огромный limit)
	posts, total, err := postSvc.GetAll(ctx, 500, -10)
	if err != nil {
		t.Fatalf("unexpected error on GetAll: %v", err)
	}
	if total != 1 || len(posts) != 1 {
		t.Fatalf("unexpected pagination result: total=%d, len=%d", total, len(posts))
	}
}

func TestCommentService_MissingPost(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	postRepo := newMockPostRepo()
	commentRepo := newMockCommentRepo()
	commentSvc := NewCommentService(commentRepo, postRepo, userRepo)

	// Создание комментария для несуществующей публикации -> ErrPostNotExists
	_, err := commentSvc.Create(ctx, 1, &model.CommentCreateRequest{
		PostID:  999,
		Content: "Comment to missing post",
	})
	if !errors.Is(err, ErrPostNotExists) {
		t.Fatalf("expected ErrPostNotExists, got %v", err)
	}
}
