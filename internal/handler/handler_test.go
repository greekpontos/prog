package handler

import (
	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/internal/service"
	"blog-api/pkg/auth"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type memUserRepo struct {
	users  map[int]*model.User
	nextID int
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: make(map[int]*model.User), nextID: 1}
}

func (r *memUserRepo) Create(ctx context.Context, u *model.User) error {
	for _, ex := range r.users {
		if ex.Email == u.Email || ex.Username == u.Username {
			return repository.ErrUserExists
		}
	}
	u.ID = r.nextID
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	r.nextID++
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *memUserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	if u, ok := r.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *memUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (r *memUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	for _, u := range r.users {
		if u.Username == username {
			cp := *u
			return &cp, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (r *memUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, err := r.GetByEmail(ctx, email)
	return err == nil, nil
}

func (r *memUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	_, err := r.GetByUsername(ctx, username)
	return err == nil, nil
}

type memPostRepo struct {
	posts  map[int]*model.Post
	nextID int
}

func newMemPostRepo() *memPostRepo {
	return &memPostRepo{posts: make(map[int]*model.Post), nextID: 1}
}

func (r *memPostRepo) Create(ctx context.Context, p *model.Post) error {
	p.ID = r.nextID
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	r.nextID++
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *memPostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	if p, ok := r.posts[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, repository.ErrPostNotFound
}

func (r *memPostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	res := make([]*model.Post, 0)
	for i := 1; i < r.nextID; i++ {
		if p, ok := r.posts[i]; ok {
			cp := *p
			res = append(res, &cp)
		}
	}
	if offset >= len(res) {
		return []*model.Post{}, nil
	}
	end := offset + limit
	if end > len(res) {
		end = len(res)
	}
	return res[offset:end], nil
}

func (r *memPostRepo) GetTotalCount(ctx context.Context) (int, error) {
	return len(r.posts), nil
}

func (r *memPostRepo) GetByAuthorID(ctx context.Context, authorID, limit, offset int) ([]*model.Post, error) {
	return []*model.Post{}, nil
}

func (r *memPostRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	return 0, nil
}

func (r *memPostRepo) Update(ctx context.Context, p *model.Post) error {
	if _, ok := r.posts[p.ID]; !ok {
		return repository.ErrPostNotFound
	}
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *memPostRepo) Delete(ctx context.Context, id int) error {
	if _, ok := r.posts[id]; !ok {
		return repository.ErrPostNotFound
	}
	delete(r.posts, id)
	return nil
}

func (r *memPostRepo) Exists(ctx context.Context, id int) (bool, error) {
	_, ok := r.posts[id]
	return ok, nil
}

type memCommentRepo struct {
	comments map[int]*model.Comment
	nextID   int
}

func newMemCommentRepo() *memCommentRepo {
	return &memCommentRepo{comments: make(map[int]*model.Comment), nextID: 1}
}

func (r *memCommentRepo) Create(ctx context.Context, c *model.Comment) error {
	c.ID = r.nextID
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt
	r.nextID++
	cp := *c
	r.comments[c.ID] = &cp
	return nil
}

func (r *memCommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	if c, ok := r.comments[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, repository.ErrCommentNotFound
}

func (r *memCommentRepo) GetByPostID(ctx context.Context, postID, limit, offset int) ([]*model.Comment, error) {
	return []*model.Comment{}, nil
}

func (r *memCommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	return 0, nil
}

func (r *memCommentRepo) GetByAuthorID(ctx context.Context, authorID, limit, offset int) ([]*model.Comment, error) {
	return []*model.Comment{}, nil
}

func (r *memCommentRepo) GetCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	return 0, nil
}

func (r *memCommentRepo) Update(ctx context.Context, c *model.Comment) error {
	return nil
}

func (r *memCommentRepo) Delete(ctx context.Context, id int) error {
	return nil
}

func setupTestRouter() http.Handler {
	jwtMgr := auth.NewJWTManager("super-secret-jwt-key-for-testing-32chars!", 24)
	userRepo := newMemUserRepo()
	postRepo := newMemPostRepo()
	commentRepo := newMemCommentRepo()

	userSvc := service.NewUserService(userRepo, jwtMgr)
	postSvc := service.NewPostService(postRepo, userRepo)
	commentSvc := service.NewCommentService(commentRepo, postRepo, userRepo)

	authH := NewAuthHandler(userSvc)
	postH := NewPostHandler(postSvc)
	commentH := NewCommentHandler(commentSvc)
	authMW := middleware.NewAuthMiddleware(jwtMgr)

	r := chi.NewRouter()
	r.Post("/api/register", authH.Register)
	r.Post("/api/login", authH.Login)
	r.Get("/api/posts", postH.GetAll)
	r.Get("/api/posts/{id}", postH.GetByID)

	r.Post("/api/posts", authMW.RequireAuth(postH.Create))
	r.Put("/api/posts/{id}", authMW.RequireAuth(postH.Update))
	r.Delete("/api/posts/{id}", authMW.RequireAuth(postH.Delete))
	r.Post("/api/posts/{postId}/comments", authMW.RequireAuth(commentH.Create))

	return r
}

func TestHTTP_EndToEndFlows(t *testing.T) {
	router := setupTestRouter()

	// 1. Доступ к защищенному маршруту без токена -> 401 Unauthorized
	reqNoAuth := httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"Test","content":"Body"}`))
	recNoAuth := httptest.NewRecorder()
	router.ServeHTTP(recNoAuth, reqNoAuth)
	if recNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", recNoAuth.Code)
	}

	// 2. Регистрация первого пользователя -> 201 Created
	regBody1 := `{"username":"user1","email":"user1@example.com","password":"password123"}`
	reqReg1 := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(regBody1))
	recReg1 := httptest.NewRecorder()
	router.ServeHTTP(recReg1, reqReg1)
	if recReg1.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d: %s", recReg1.Code, recReg1.Body.String())
	}
	var tokenResp1 model.TokenResponse
	_ = json.NewDecoder(recReg1.Body).Decode(&tokenResp1)

	// 3. Повторная регистрация с тем же email -> 409 Conflict
	reqRegDup := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(regBody1))
	recRegDup := httptest.NewRecorder()
	router.ServeHTTP(recRegDup, reqRegDup)
	if recRegDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate register, got %d", recRegDup.Code)
	}

	// 4. Регистрация второго пользователя -> 201 Created
	regBody2 := `{"username":"user2","email":"user2@example.com","password":"password123"}`
	reqReg2 := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(regBody2))
	recReg2 := httptest.NewRecorder()
	router.ServeHTTP(recReg2, reqReg2)
	var tokenResp2 model.TokenResponse
	_ = json.NewDecoder(recReg2.Body).Decode(&tokenResp2)

	// 5. Создание поста первым пользователем -> 201 Created
	postBody := `{"title":"Hello World","content":"My first blog post"}`
	reqCreatePost := httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(postBody))
	reqCreatePost.Header.Set("Authorization", "Bearer "+tokenResp1.Token)
	recCreatePost := httptest.NewRecorder()
	router.ServeHTTP(recCreatePost, reqCreatePost)
	if recCreatePost.Code != http.StatusCreated {
		t.Fatalf("expected 201 on post create, got %d", recCreatePost.Code)
	}
	var createdPost model.Post
	_ = json.NewDecoder(recCreatePost.Body).Decode(&createdPost)

	// 6. Попытка второго пользователя изменить чужой пост -> 403 Forbidden
	updateBody := `{"title":"Edited","content":"By user 2"}`
	reqUpdateForbidden := httptest.NewRequest(http.MethodPut, "/api/posts/"+strconv.Itoa(createdPost.ID), bytes.NewBufferString(updateBody))
	reqUpdateForbidden.Header.Set("Authorization", "Bearer "+tokenResp2.Token)
	recUpdateForbidden := httptest.NewRecorder()
	router.ServeHTTP(recUpdateForbidden, reqUpdateForbidden)
	if recUpdateForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when updating foreign post, got %d", recUpdateForbidden.Code)
	}

	// 7. Обращение к несуществующему посту -> 404 Not Found
	reqNotFound := httptest.NewRequest(http.MethodGet, "/api/posts/9999", nil)
	recNotFound := httptest.NewRecorder()
	router.ServeHTTP(recNotFound, reqNotFound)
	if recNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing post, got %d", recNotFound.Code)
	}

	// 8. Создание комментария для несуществующего поста -> 404 Not Found
	commentBody := `{"content":"Nice post!"}`
	reqCommentMissingPost := httptest.NewRequest(http.MethodPost, "/api/posts/9999/comments", bytes.NewBufferString(commentBody))
	reqCommentMissingPost.Header.Set("Authorization", "Bearer "+tokenResp1.Token)
	recCommentMissingPost := httptest.NewRecorder()
	router.ServeHTTP(recCommentMissingPost, reqCommentMissingPost)
	if recCommentMissingPost.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when commenting on missing post, got %d", recCommentMissingPost.Code)
	}

	// 9. Проверка граничных параметров пагинации -> 200 OK, limit ограничен до 100, offset >= 0
	reqPage := httptest.NewRequest(http.MethodGet, "/api/posts?limit=500&offset=-5", nil)
	recPage := httptest.NewRecorder()
	router.ServeHTTP(recPage, reqPage)
	if recPage.Code != http.StatusOK {
		t.Fatalf("expected 200 on pagination, got %d", recPage.Code)
	}
	var pageResp map[string]interface{}
	_ = json.NewDecoder(recPage.Body).Decode(&pageResp)
	if int(pageResp["limit"].(float64)) != 100 || int(pageResp["offset"].(float64)) != 0 {
		t.Fatalf("expected normalized limit=100 and offset=0, got %+v", pageResp)
	}
}
