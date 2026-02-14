package url

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func (a *App) buildHealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status": "ok"}`))
		if err != nil {
			writeJSONError(w, 500, "internal error")
		}
	}
}

func (a *App) buildIndexServeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		countUpTotalRequestCounter(r.RequestURI)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Prefer current working directory (common during `go run .` from project root)
		wd, _ := os.Getwd()
		p1 := filepath.Join(wd, "index.html")
		if _, err := os.Stat(p1); err == nil {
			http.ServeFile(w, r, p1)
			return
		}

		http.Error(w, "index.html not found", http.StatusNotFound)
	}
}

func (a *App) buildShortURLCreationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request ShortURLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			countUpShortURLCreationCounter("failed")
			writeJSONError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		err := verifyURL(request.OriginalURL)
		if err != nil {
			countUpShortURLCreationCounter("failed")
			writeJSONError(w, http.StatusBadRequest, "invalid url")
			return
		}

		if strings.HasPrefix(request.OriginalURL, "https://") {
			request.OriginalURL = "https://" + request.OriginalURL
		}

		// TODO think, how to test
		code, err := newCode(8)
		if err != nil {
			countUpShortURLCreationCounter("failed")
			a.lgr.Error("can't generate short code", "url", request.OriginalURL, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "something went wrong")
			return
		}

		shortURL := ShortURL{
			OriginalURL: request.OriginalURL,
			ShortCode:   code,
		}

		// TODO replace to interface
		if err = a.db.Create(&shortURL).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				var regeneratedCode string
				regeneratedCode, err = newCode(8)
				if err != nil {
					countUpShortURLCreationCounter("failed")
					writeJSONError(w, http.StatusInternalServerError, "can't generate short code again")
					return
				}

				shortURL.ShortCode = regeneratedCode
				if err = a.db.Create(&shortURL).Error; err != nil {
					countUpShortURLCreationCounter("failed")
					a.lgr.Error("can't create short url to DB", "url", request.OriginalURL, "error", err)
					writeJSONError(w, http.StatusInternalServerError, "something went wrong")
					return
				}
				countUpShortURLCreationCounter("success")
				resp := ShortCodeResponse{
					ShortCode: shortURL.ShortCode,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			countUpShortURLCreationCounter("failed")
			a.lgr.Error("can't create short url", "url", request.OriginalURL, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "something went wrong")
			return
		}

		countUpShortURLCreationCounter("success")
		resp := ShortCodeResponse{
			ShortCode: shortURL.ShortCode,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func verifyURL(url string) error {
	if url == "" {
		return errors.New("url is empty")
	}
	if len(url) > 2048 {
		return errors.New("url is too long")
	}

	return nil
}

func (a *App) buildGettingShortURLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		countUpTotalRequestCounter(r.RequestURI)

		code := r.PathValue("code")
		// Reject codes that contain a file extension (e.g. ".html", ".png")
		if strings.Contains(code, ".") {
			writeJSONError(w, http.StatusBadRequest, "invalid short code")
			return
		}
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
