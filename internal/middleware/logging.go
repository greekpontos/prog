package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// LoggingMiddleware предоставляет middleware для логирования, CORS, RequestID и перехвата паник
type LoggingMiddleware struct {
	logger *log.Logger
}

// NewLoggingMiddleware создает новый экземпляр middleware логирования
func NewLoggingMiddleware(logger *log.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Logger логирует метод, путь, IP клиента, статус ответа и длительность обработки запроса
func (m *LoggingMiddleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := newResponseWriter(w)

		next.ServeHTTP(wrapped, r)

		m.logger.Printf("[%s] %s %s - %d (%v)",
			getClientIP(r),
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start),
		)
	})
}

// Recovery перехватывает паники в обработчиках и возвращает 500 Internal Server Error
func (m *LoggingMiddleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Printf("PANIC RECOVERED: %v\n%s", err, debug.Stack())
				writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS устанавливает заголовки Cross-Origin Resource Sharing и обрабатывает OPTIONS запросы
func (m *LoggingMiddleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequestID добавляет уникальный заголовок X-Request-ID к каждому запросу
func (m *LoggingMiddleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

// getClientIP определяет реальный IP-адрес клиента из заголовков прокси или RemoteAddr
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		firstIP, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(firstIP)
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// responseWriter оборачивает стандартный http.ResponseWriter для перехвата кода ответа
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader сохраняет HTTP статус-код ответа
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

// Write записывает тело ответа и фиксирует статус 200 OK по умолчанию
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// newResponseWriter создает обертку для ResponseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}
}
