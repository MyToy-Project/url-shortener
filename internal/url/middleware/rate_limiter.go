package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"url-shortener/internal/url/handler"

	"golang.org/x/time/rate"
)

// clientLimiter stores a per-client rate limiter and its last-seen timestamp.
type clientLimiter struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

var (
	clientLimitersMu sync.Mutex
	clientLimiters   = map[string]*clientLimiter{}
)

// RateLimitByIP returns a chi middleware that limits requests per client IP.
// rpm: requests per minute
// burst: allowed burst size
func RateLimitByIP(rpm int, burst int) func(http.Handler) http.Handler {
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
				// TODO create counter metric
				slog.Default().Warn("rate limit exceeded", "ip", ip)
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

	_ = json.NewEncoder(w).Encode(handler.ErrorResponse{
		Message: message,
	})
}
