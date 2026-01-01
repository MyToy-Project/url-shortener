package url

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"time"

	"gorm.io/gorm"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type ShortURL struct {
	UrlID       uint `gorm:"primaryKey"`
	OriginalURL string
	ShortCode   string `gorm:"uniqueIndex"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type ShortURLRequest struct {
	OriginalURL string `json:"original_url"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func (a *App) registerRoutes() {
	a.r.Post("/short-url", a.buildShortURLCreation())
	a.r.Get("/{code}", a.buildGettingShortURL())
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
	}
}

func newCode(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	out := make([]byte, n)
	maxLen := big.NewInt(int64(len(alphabet)))

	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, maxLen)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[x.Int64()]
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
