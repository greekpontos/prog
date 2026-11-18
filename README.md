# Blog API

REST API для блог-платформы на языке Go с использованием PostgreSQL, слоистой архитектуры (`Handler -> Service -> Repository`) и JWT-аутентификации.

## Реализованная функциональность

### Обязательные возможности
- Регистрация и аутентификация пользователей с хешированием паролей через `bcrypt`.
- Выдача и проверка JWT-токенов (алгоритм `HS256`, контроль срока действия).
- Полный CRUD для публикаций (`/api/posts`) с проверкой авторства при изменении и удалении.
- Полный CRUD для комментариев (`/api/posts/{postId}/comments`, `/api/comments/{id}`) с проверкой существования поста и прав автора.
- SQL-пагинация (`limit`, `offset`) с возвратом метаданных (`count`, `total`, `limit`, `offset`) и стабильной сортировкой по дате и ID.
- Автоматическое применение миграций PostgreSQL при старте приложения с учётом применённых версий (`schema_migrations`).
- Единый JSON-формат ответов об ошибках и структурированное логирование запросов.

### Дополнительные возможности
- Фильтрация публикаций по автору (`GET /api/posts/author/{authorID}`).
- Middleware восстановления после паник (`Recovery`), обработки CORS и генерации `X-Request-ID`.
- Корректное завершение работы сервера (graceful shutdown) с закрытием пула соединений БД.

## Технологический стек
- **Go 1.26** — основной язык разработки
- **Chi (`go-chi/chi/v5`)** — HTTP-маршрутизатор и middleware
- **PostgreSQL 15** — реляционная база данных (`://github.com`)
- **JWT (`golang-jwt/jwt/v5`)** — аутентификация и авторизация пользователей
- **bcrypt (`golang.org/x/crypto/bcrypt`)** — безопасное хеширование паролей
- **Docker / Docker Compose** — локальное развёртывание PostgreSQL и Adminer

## Переменные окружения

Настройки задаются в файле `.env` (образец находится в `.env.example`):

| Переменная | Описание | Значение по умолчанию |
| :--- | :--- | :--- |
| `SERVER_HOST` | Хост HTTP-сервера | `localhost` |
| `SERVER_PORT` | Порт HTTP-сервера | `8080` |
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_PORT` | Порт PostgreSQL | `5432` |
| `DB_USER` | Пользователь БД | `bloguser` |
| `DB_PASSWORD` | Пароль пользователя БД | `blogpassword` |
| `DB_NAME` | Имя базы данных | `blogdb` |
| `DB_SSLMODE` | Режим SSL подключения к БД | `disable` |
| `JWT_SECRET` | Секретный ключ подписи JWT (минимум 32 символа) | `super-secret-jwt-key-at-least-32-chars-long!` |
| `JWT_EXPIRY_HOURS` | Время жизни JWT-токена в часах | `24` |

## Последовательность запуска

1. **Создайте файл `.env` из примера:**
   ```bash
   cp .env.example .env
   ```

2. **Запустите PostgreSQL в Docker:**
   ```bash
   docker-compose up -d postgres
   ```

3. **Установите зависимости и запустите приложение:**
   ```bash
   go mod download
   go run cmd/api/main.go
   ```
   При запуске автоматически выполняется миграция схемы БД, после чего API становится доступно на `http://localhost:8080`.

## Правила передачи JWT

Для доступа к защищённым эндпоинтам необходимо передавать токен, полученный при регистрации или входе, в HTTP-заголовке:
```http
Authorization: Bearer <ваш_jwt_токен>
```

## Фактические маршруты API

### Публичные эндпоинты
- `GET /api/health` — проверка состояния сервиса
- `POST /api/register` — регистрация нового пользователя
- `POST /api/login` — вход пользователя и получение JWT-токена
- `GET /api/posts?limit=10&offset=0` — список постов с пагинацией (`limit` от 1 до 100, `offset >= 0`)
- `GET /api/posts/{id}` — получение поста по ID
- `GET /api/posts/author/{authorID}?limit=10&offset=0` — получение постов конкретного автора
- `GET /api/posts/{postId}/comments?limit=20&offset=0` — получение комментариев к посту
- `GET /api/comments/{id}` — получение комментария по ID

### Защищённые эндпоинты (требуют заголовок `Authorization: Bearer <token>`)
- `POST /api/posts` — создание нового поста
- `PUT /api/posts/{id}` — редактирование поста (только автор)
- `DELETE /api/posts/{id}` — удаление поста (только автор)
- `POST /api/posts/{postId}/comments` — добавление комментария к посту
- `POST /api/comments` — добавление комментария (с передачей `post_id` в JSON-теле)
- `PUT /api/comments/{id}` — редактирование комментария (только автор)
- `DELETE /api/comments/{id}` — удаление комментария (только автор)

## Форматы запросов и ответов

### Регистрация (`POST /api/register`)
Запрос:
```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "secretPassword123"
}
```
Ответ (`201 Created`):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-11-19T22:00:00Z",
  "user": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2026-11-18T22:00:00Z"
  }
}
```

### Создание поста (`POST /api/posts`)
Запрос:
```json
{
  "title": "Заголовок публикации",
  "content": "Основной текст публикации"
}
```
Ответ (`201 Created`):
```json
{
  "id": 1,
  "title": "Заголовок публикации",
  "content": "Основной текст публикации",
  "author_id": 1,
  "author": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2026-11-18T22:00:00Z"
  },
  "created_at": "2026-11-18T22:05:00Z",
  "updated_at": "2026-11-18T22:05:00Z"
}
```

### Список постов с пагинацией (`GET /api/posts?limit=10&offset=0`)
Ответ (`200 OK`):
```json
{
  "posts": [
    {
      "id": 1,
      "title": "Заголовок публикации",
      "content": "Основной текст публикации",
      "author_id": 1,
      "created_at": "2026-11-18T22:05:00Z",
      "updated_at": "2026-11-18T22:05:00Z"
    }
  ],
  "count": 1,
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

### Формат ошибки
Все ошибки возвращаются в едином JSON-формате:
```json
{
  "error": "Forbidden",
  "message": "You can only update your own posts"
}
```

## Основные HTTP-статусы
- `200 OK` — успешное получение данных, вход в систему или обновление записи
- `201 Created` — успешная регистрация пользователя, создание поста или комментария
- `204 No Content` — успешное удаление поста или комментария
- `400 Bad Request` — некорректный JSON или ошибка валидации входных данных
- `401 Unauthorized` — отсутствует, недействителен или просрочен JWT-токен, либо неверные учётные данные при входе
- `403 Forbidden` — попытка изменить или удалить чужой пост или комментарий
- `404 Not Found` — запрашиваемый пользователь, пост или комментарий не найден
- `409 Conflict` — пользователь с таким `email` или `username` уже зарегистрирован
- `500 Internal Server Error` — внутренняя ошибка сервера или базы данных

## Тестирование и проверка качества кода

Запуск форматирования, статического анализа и автоматических тестов:
```bash
go fmt ./...
go vet ./...
go test -v ./...
```
