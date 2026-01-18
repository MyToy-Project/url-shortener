package url

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCountUpNumberOfRequestCounter(t *testing.T) {
	prev, _ := testutil.GatherAndCount(prometheus.DefaultGatherer, totalRequestName)
	countUpTotalRequestCounter("/")
	curr, _ := testutil.GatherAndCount(prometheus.DefaultGatherer, totalRequestName)
	assert.Equal(t, prev, curr-1)
}

func TestCountupShortURLCreationCounter(t *testing.T) {
	prev, _ := testutil.GatherAndCount(prometheus.DefaultGatherer, shortURLCreationName)
	countUpShortURLCreationCounter("success")
	curr, _ := testutil.GatherAndCount(prometheus.DefaultGatherer, shortURLCreationName)
	assert.Equal(t, prev, curr-1)
}
