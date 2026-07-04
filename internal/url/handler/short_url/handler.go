package short_url

import (
	"encoding/json"
	"net/http"
	"url-shortener/internal/url"
	"url-shortener/internal/url/handler"
)

type ShortURLHandler struct {
	svc URLService
}

type URLService interface {
	GenerateShortURL(originalURL string) (string, error)
	GetOriginalURL(code string) (string, error)
}

// ShortURLRequest represents a request to create a shortened URL
type ShortURLRequest struct {
	OriginalURL string `json:"original_url"`
}

// ShortCodeResponse represents a response containing the shortened URL
type ShortCodeResponse struct {
	ShortCode string `json:"short_code"`
}

func NewHandler(svc URLService) *ShortURLHandler {
	return &ShortURLHandler{
		svc: svc,
	}
}

func (h *ShortURLHandler) HandleShortURLCreation(w http.ResponseWriter, r *http.Request) {
	var request ShortURLRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		url.CountUpShortURLCreationCounter("failed")
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	generateShortURL, err := h.svc.GenerateShortURL(request.OriginalURL)

	if err != nil {
		url.CountUpShortURLCreationCounter("failed")
		writeJSONError(w, http.StatusBadRequest, "Failed to generate short_url")
	}

	resp := ShortCodeResponse{
		ShortCode: generateShortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)

}

func (h *ShortURLHandler) HandleGettingOriginalURL(w http.ResponseWriter, r *http.Request) {
	url.CountUpTotalRequestCounter(r.RequestURI)
	code := r.PathValue("code")
	originalURL, err := h.svc.GetOriginalURL(code)

	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to get short url")
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
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
