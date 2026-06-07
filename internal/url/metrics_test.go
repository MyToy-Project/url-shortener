package url

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func setupTestRegistry(t *testing.T) *prometheus.Registry {
	t.Helper()

	reg := prometheus.NewRegistry()
	totalRequestCounter = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: totalRequestName,
			Help: totalRequestDescription,
		},
		[]string{"type"},
	)
	shortURLCreationCounter = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: shortURLCreationName,
			Help: shortURLCreationDescription,
		},
		[]string{"state"},
	)
	return reg
}

func TestCountUpNumberOfRequestCounter(t *testing.T) {
	reg := setupTestRegistry(t)
	prev, _ := testutil.GatherAndCount(reg, totalRequestName)
	CountUpTotalRequestCounter("/")
	curr, _ := testutil.GatherAndCount(reg, totalRequestName)
	assert.Equal(t, prev, curr-1)
}

func TestCountupShortURLCreationCounter(t *testing.T) {
	reg := setupTestRegistry(t)
	prev, _ := testutil.GatherAndCount(reg, shortURLCreationName)
	CountUpShortURLCreationCounter("success")
	curr, _ := testutil.GatherAndCount(reg, shortURLCreationName)
	assert.Equal(t, prev, curr-1)
}
