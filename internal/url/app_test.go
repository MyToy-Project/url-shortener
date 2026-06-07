package url

import (
	"testing"
	"url-shortener/internal/url/handler"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNewApp(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"))
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&handler.ShortURL{}); err != nil {
		t.Fatal(err)
	}
	app := NewApp(db)

	assert.IsType(t, &App{}, app)
}
