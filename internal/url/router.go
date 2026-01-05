package url

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/cors"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// ShortURL represents a shortened URL in the database
type ShortURL struct {
	UrlID       uint `gorm:"primaryKey"`
	OriginalURL string
	ShortCode   string `gorm:"uniqueIndex"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ShortURLRequest represents a request to create a shortened URL
type ShortURLRequest struct {
	OriginalURL string `json:"original_url"`
}

// ShortCodeResponse represents a response containing the shortened URL
type ShortCodeResponse struct {
	ShortCode string `json:"short_code"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
}

func (a *App) registerRoutes() {
	a.r.Use(cors.Handler(
		cors.Options{
			AllowedOrigins: []string{"http://localhost:5500", "http://127.0.0.1:5500"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
			MaxAge:         300,
		},
	))
	a.r.Get("/", a.serveIndex())

	// Rate limit only the shorten endpoint (per IP)
	// Example: 30 requests per minute with a burst of 1
	a.r.With(rateLimitByIP(30, 1)).Post("/api/shorten", a.buildShortURLCreation())
	a.r.Get("/{code}", a.buildGettingShortURL())
}

func (a *App) serveIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Prefer current working directory (common during `go run .` from project root)
		wd, _ := os.Getwd()
		fmt.Print(wd)
		p1 := filepath.Join(wd, "index.html")
		if _, err := os.Stat(p1); err == nil {
			http.ServeFile(w, r, p1)
			return
		}

		http.Error(w, "index.html not found", http.StatusNotFound)
	}
}

func (a *App) buildShortURLCreation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request ShortURLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		if request.OriginalURL == "" {
			writeJSONError(w, http.StatusBadRequest, "original url is required")
			return
		}

		code, err := newCode(8)
		if err != nil {
			a.lgr.Error("can't generate short code", "url", request.OriginalURL, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "something went wrong")
			return
		}

		shortURL := ShortURL{
			OriginalURL: request.OriginalURL,
			ShortCode:   code,
		}

		if err = a.db.Create(&shortURL).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				var regeneratedCode string
				regeneratedCode, err = newCode(8)
				if err == nil {
					writeJSONError(w, http.StatusInternalServerError, "can't generate short code again")
					return
				}

				shortURL.ShortCode = regeneratedCode
				if err = a.db.Create(&shortURL).Error; err != nil {
					a.lgr.Error("can't insert short url to DB", "url", request.OriginalURL, "error", err)
					writeJSONError(w, http.StatusInternalServerError, "something went wrong")
					return
				}
			}
			a.lgr.Error("can't create short url", "url", request.OriginalURL, "error", err)
		}

		resp := ShortCodeResponse{
			ShortCode: shortURL.ShortCode,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func newCode(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	out := make([]byte, n)
	maxLen := big.NewInt(int64(len(base62)))

	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, maxLen)
		if err != nil {
			return "", err
		}
		out[i] = base62[x.Int64()]
	}
	return string(out), nil
}

func (a *App) buildGettingShortURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")
		if code == "" {
			writeJSONError(w, http.StatusBadRequest, "code is required")
			return
		}
		var shortURL ShortURL
		if err := a.db.Where(&ShortURL{ShortCode: code}).First(&shortURL).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeJSONError(w, http.StatusNotFound, "short url not found")
				return
			}
			a.lgr.Error("can't find short url", "code", code, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "something went wrong")
			return
		}

		http.Redirect(w, r, shortURL.OriginalURL, http.StatusFound)
	}
}

// clientLimiter stores a per-client rate limiter and its last-seen timestamp.
type clientLimiter struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

var (
	clientLimitersMu sync.Mutex
	clientLimiters   = map[string]*clientLimiter{}
)

// rateLimitByIP returns a chi middleware that limits requests per client IP.
// rpm: requests per minute
// burst: allowed burst size
func rateLimitByIP(rpm int, burst int) func(http.Handler) http.Handler {
	// If disabled/misconfigured, do nothing.
	if rpm <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if burst <= 0 {
		burst = 1
	}

	// Periodically cleanup old entries to avoid unbounded memory growth.
	// This is a cheap, best-effort cleanup; no goroutines are started.
	cleanupAfter := 10 * time.Minute

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			slog.Default().Info("rateLimitByIP", "ip", ip)
			clientLimitersMu.Lock()
			// Best-effort cleanup on request path.
			now := time.Now()
			for k, v := range clientLimiters {
				if now.Sub(v.lastSeen) > cleanupAfter {
					delete(clientLimiters, k)
				}
			}

			cl, ok := clientLimiters[ip]
			if !ok {
				// Convert rpm to an event interval.
				// Example: 30 rpm => 1 request per 2 seconds.
				interval := time.Minute / time.Duration(rpm)
				cl = &clientLimiter{
					lim:      rate.NewLimiter(rate.Every(interval), burst),
					lastSeen: now,
				}
				clientLimiters[ip] = cl
			} else {
				cl.lastSeen = now
			}
			allow := cl.lim.Allow()
			clientLimitersMu.Unlock()

			if !allow {
				writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts a best-effort client IP.
// If behind a trusted proxy, ensure your proxy sets X-Forwarded-For or X-Real-IP.
func clientIP(r *http.Request) string {
	// X-Forwarded-For may contain multiple IPs: client, proxy1, proxy2, ...
	slog.Default().Info("all ips", "ips", r.Header)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	xrip := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xrip != "" {
		return xrip
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	// Fallback
	return strings.TrimSpace(r.RemoteAddr)
}

func writeJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Message: message,
	})
}
