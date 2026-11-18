package main

import (
	"blog-api/internal/handler"
	"blog-api/internal/middleware"
	"blog-api/internal/repository"
	"blog-api/internal/service"
	"blog-api/pkg/auth"
	"blog-api/pkg/database"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем переменные окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	// Загружаем и проверяем конфигурацию приложения
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Формируем параметры подключения к PostgreSQL
	dbCfg := database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}

	// Устанавливаем соединение с базой данных
	db, err := database.NewPostgresDB(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("Error closing database connection: %v", err)
		} else {
			log.Println("Database connection closed")
		}
	}()

	// Выполняем автоматические миграции при запуске
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("Database migrations applied successfully")

	// Инициализируем менеджер JWT токенов
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHours)

	// Инициализируем слой репозиториев
	userRepo := repository.NewUserRepo(db)
	postRepo := repository.NewPostRepo(db)
	commentRepo := repository.NewCommentRepo(db)

	// Инициализируем слой сервисов (бизнес-логика)
	userService := service.NewUserService(userRepo, jwtManager)
	postService := service.NewPostService(postRepo, userRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, userRepo)

	// Инициализируем HTTP-хендлеры
	authHandler := handler.NewAuthHandler(userService)
	postHandler := handler.NewPostHandler(postService)
	commentHandler := handler.NewCommentHandler(commentService)

	// Инициализируем middleware
	logger := log.New(os.Stdout, "[BLOG-API] ", log.LstdFlags)
	loggingMW := middleware.NewLoggingMiddleware(logger)
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// Настраиваем маршрутизатор Chi
	router := chi.NewRouter()

	// Подключаем глобальные middleware
	router.Use(loggingMW.Recovery)
	router.Use(loggingMW.CORS)
	router.Use(loggingMW.RequestID)
	router.Use(loggingMW.Logger)

	// Эндпоинт проверки состояния сервиса
	router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"blog-api"}`))
	})

	// Публичные маршруты (доступны без аутентификации)
	router.Post("/api/register", authHandler.Register)
	router.Post("/api/login", authHandler.Login)

	router.Get("/api/posts", postHandler.GetAll)
	router.Get("/api/posts/{id}", postHandler.GetByID)
	router.Get("/api/posts/author/{authorID}", postHandler.GetByAuthor)
	router.Get("/api/posts/{postId}/comments", commentHandler.GetByPost)
	router.Get("/api/comments/{id}", commentHandler.GetByID)

	// Защищенные маршруты (требуют валидный JWT токен)
	router.Post("/api/posts", authMW.RequireAuth(postHandler.Create))
	router.Put("/api/posts/{id}", authMW.RequireAuth(postHandler.Update))
	router.Delete("/api/posts/{id}", authMW.RequireAuth(postHandler.Delete))

	router.Post("/api/posts/{postId}/comments", authMW.RequireAuth(commentHandler.Create))
	router.Post("/api/comments", authMW.RequireAuth(commentHandler.Create))
	router.Put("/api/comments/{id}", authMW.RequireAuth(commentHandler.Update))
	router.Delete("/api/comments/{id}", authMW.RequireAuth(commentHandler.Delete))

	// Настраиваем и запускаем HTTP-сервер с поддержкой корректного завершения
	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Ожидаем сигнал завершения процесса или критическую ошибку сервера
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server failed: %v", err)
	case sig := <-quit:
		log.Printf("Shutting down server (signal: %v)...", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited properly")
}

// Config представляет конфигурацию приложения из переменных окружения
type Config struct {
	ServerHost      string
	ServerPort      int
	DBHost          string
	DBPort          int
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	JWTSecret       string
	JWTExpiryHours  int
	CacheTTLMinutes int
}

// loadConfig загружает и валидирует настройки приложения из переменных окружения
func loadConfig() (*Config, error) {
	cfg := &Config{
		ServerHost:      getEnv("SERVER_HOST", "localhost"),
		ServerPort:      getEnvAsInt("SERVER_PORT", 8080),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnvAsInt("DB_PORT", 5432),
		DBUser:          getEnv("DB_USER", "bloguser"),
		DBPassword:      getEnv("DB_PASSWORD", "blogpassword"),
		DBName:          getEnv("DB_NAME", "blogdb"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		JWTSecret:       getEnv("JWT_SECRET", "super-secret-jwt-key-at-least-32-chars-long!"),
		JWTExpiryHours:  getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		CacheTTLMinutes: getEnvAsInt("CACHE_TTL_MINUTES", 5),
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters long for cryptographic security")
	}

	return cfg, nil
}

// getEnv возвращает строковое значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt возвращает числовое значение переменной окружения или значение по умолчанию
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
