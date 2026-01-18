package url

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	totalRequestName        = `total_request`
	totalRequestDescription = `Number of total requests per type`

	shortURLCreationName        = `short_url_creation`
	shortURLCreationDescription = `Number of total short URL creation`
)

var (
	totalRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: totalRequestName,
			Help: totalRequestDescription,
		},
		[]string{"type"},
	)

	shortURLCreationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: shortURLCreationName,
			Help: shortURLCreationDescription,
		},
		[]string{"state"},
	)
)

func countUpTotalRequestCounter(typeName string) {
	totalRequestCounter.With(prometheus.Labels{"type": typeName}).Inc()
}

func countUpShortURLCreationCounter(state string) {
	shortURLCreationCounter.With(prometheus.Labels{"state": state}).Inc()
}
