package short_url

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type FakeURLService struct{}

func NewFakeURLService() *FakeURLService {
	return &FakeURLService{}
}

func (*FakeURLService) GenerateShortURL(originalURL string) (string, error) {
	if originalURL == "" || originalURL == "https://fail.com" {
		return "", errors.New("fail")
	}
	return "https://test_url.com", nil
}

func (s *FakeURLService) GetOriginalURL(code string) (string, error) {
	if code == "success" {
		return "original_url", nil
	}
	return "", errors.New("not found")
}

func TestNewHandler(t *testing.T) {
	handler := NewHandler(NewFakeURLService())
	assert.IsTypef(t, &ShortURLHandler{}, handler, "NewHandler returned wrong handler type")
}

func TestShortURLHandler_HandleGettingOriginalURL(t *testing.T) {
	type expected struct {
		statusCode       int
		expectedLocation string
	}
	tests := []struct {
		name          string
		svc           URLService
		targetPathVar string
		expected      expected
	}{
		{
			name:          "Success getting original URL",
			svc:           NewFakeURLService(),
			targetPathVar: "success",
			expected: expected{
				statusCode:       http.StatusFound,
				expectedLocation: "/original_url",
			},
		},
		{
			name:          "Fail getting original URL",
			svc:           NewFakeURLService(),
			targetPathVar: "/random_fail_value",
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/{code}", nil)
			r.SetPathValue("code", tt.targetPathVar)
			w := httptest.NewRecorder()
			h := &ShortURLHandler{
				svc: tt.svc,
			}
			h.HandleGettingOriginalURL(w, r)
			result := w.Result()
			assert.Equal(t, tt.expected.statusCode, result.StatusCode)
			assert.Equal(t, result.Header.Get("Location"), tt.expected.expectedLocation)
		})
	}
}

func TestShortURLHandler_HandleShortURLCreation(t *testing.T) {
	type expected struct {
		statusCode int
		shortURL   string
	}
	tests := []struct {
		name     string
		svc      URLService
		body     map[string]any
		expected expected
	}{
		{
			name: "Success creating short_url",
			svc:  NewFakeURLService(),
			body: map[string]any{"original_url": "https://naver.com"},
			expected: expected{
				statusCode: http.StatusCreated,
				shortURL:   "https://test_url.com",
			},
		},
		{
			name: "Failed creating short_url",
			svc:  NewFakeURLService(),
			body: map[string]any{"original_url": ""},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb, _ := json.Marshal(tt.body)
			r := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(mb))
			w := httptest.NewRecorder()
			response := &ShortCodeResponse{}
			h := &ShortURLHandler{
				svc: tt.svc,
			}

			h.HandleShortURLCreation(w, r)

			result := w.Result()
			bb, _ := io.ReadAll(result.Body)
			_ = json.Unmarshal(bb, &response)
			assert.Equal(t, tt.expected.statusCode, result.StatusCode)
			assert.Equal(t, tt.expected.shortURL, response.ShortCode)
		})
	}
}
