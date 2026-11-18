package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config содержит параметры подключения к PostgreSQL
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// GetDSN формирует строку подключения к PostgreSQL
func GetDSN(cfg Config) string {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslMode,
	)
}

// NewPostgresDB создает новое подключение к PostgreSQL и настраивает пул соединений
func NewPostgresDB(cfg Config) (*sql.DB, error) {
	dsn := GetDSN(cfg)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем реальную доступность базы данных с таймаутом через PingContext
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настраиваем параметры пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// Migrate выполняет автоматические миграции базы данных с учетом уже примененных версий
func Migrate(db *sql.DB) error {
	// Создаем таблицу для учета примененных миграций
	const createMigrationsTable = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`

	if _, err := db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	const migrationVersion = "001_init_schema"

	// Проверяем, была ли миграция уже применена
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", migrationVersion).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}
	if exists {
		return nil
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(200) NOT NULL,
			content TEXT NOT NULL,
			author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS comments (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id);`,
		`CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id);`,
	}

	// Выполняем миграцию и запись о ней в рамках единой транзакции
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	for _, q := range queries {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("failed to execute migration query: %w", err)
		}
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migrationVersion); err != nil {
		return fmt.Errorf("failed to record migration version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// CheckConnection проверяет активное соединение с базой данных
func CheckConnection(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// Close корректно закрывает соединение с базой данных
func Close(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// TestConnection выполняет простой тестовый запрос к БД
func TestConnection(db *sql.DB) error {
	var res int
	return db.QueryRow("SELECT 1").Scan(&res)
}
