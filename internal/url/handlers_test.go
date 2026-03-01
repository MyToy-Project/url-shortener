package url

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func createTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	shortURL := ShortURL{
		OriginalURL: "https://google.com",
		ShortCode:   "test1",
	}
	db.Create(&shortURL)
}

func cleanTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&ShortURL{})
}

func newTestApp(t *testing.T) *App {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to database %v", err)
	}

	if err = db.AutoMigrate(&ShortURL{}); err != nil {
		t.Fatalf("failed to migrate short url %v", err)
	}

	lgr := slog.New(slog.NewTextHandler(io.Discard, nil))

	return &App{
		db:  db,
		lgr: lgr,
	}
}

func TestApp_buildIndexServeHandler_ServesIndexHTML(t *testing.T) {
	a := newTestApp(t)

	origWD, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "index.html"), []byte("<h1>hello</hi>"), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	h := a.buildIndexServeHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "<h1>hello</hi>" {
		t.Fatalf("unexpected response body: %q", rr.Body.String())
	}
}

func TestApp_buildIndexServeHandler_NotFound(t *testing.T) {
	a := newTestApp(t)

	h := a.buildIndexServeHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestApp_buildShortURLCreationHandler(t *testing.T) {
	longURL := createLongURL()

	tests := []struct {
		name        string
		requestBody string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "when everything works, will create short code for shortened-url",
			requestBody: `{"original_url":"https://example.com"}`,
			wantStatus:  http.StatusCreated,
			wantBody:    "short_code",
		},
		{
			name:        "when it requests with invalid request body",
			requestBody: `{"original_url":"https://google.co}`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    "invalid json body",
		},
		{
			name:        "when it requests with empty url",
			requestBody: `{"original_url":""}`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    "invalid url",
		},
		{
			name:        "when it requests with long url",
			requestBody: `{"original_url": "` + longURL + `"}`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    "invalid url",
		},
		{
			name:        "when it requests with url contains white space",
			requestBody: `{"original_url": "https://exam ple.com"}`,
			wantStatus:  http.StatusBadRequest,
			wantBody:    "invalid url",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestApp(t)

			h := a.buildShortURLCreationHandler()

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Fatalf("expected '%s' in response, got %q", tt.wantBody, rr.Body.String())
			}
		})
	}
}

func TestApp_buildGettingShortURLHandler(t *testing.T) {
	a := newTestApp(t)
	t.Cleanup(func() {
		cleanTestData(t, a.db)
	})

	tests := []struct {
		name       string
		code       string
		wantStatus int
		location   string
	}{
		{
			name:       "when everything works, it will return short code for shortened-url",
			code:       "test1",
			wantStatus: http.StatusFound,
			location:   "https://google.com",
		},
		{
			name:       "when short code not exists, it will return error",
			code:       "test2",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "when short code has invalid letter, it will return error",
			code:       "test.1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "when short code is empty, it will return error",
			code:       "",
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createTestData(t, a.db)

			h := a.buildGettingShortURLHandler()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetPathValue("code", tt.code)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Equal(t, tt.location, rr.Header().Get("Location"))

			cleanTestData(t, a.db)
		})
	}
}

func createLongURL() string {
	return "https://google.com" + strings.Repeat("a", 2048-17)
}
