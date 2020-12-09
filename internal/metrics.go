package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	PrometheusNamespace = "rdsys_backend"
)

type Metrics struct {
	TestedResources *prometheus.GaugeVec
	Resources       *prometheus.GaugeVec
	Requests        *prometheus.CounterVec
}

// InitMetrics initialises our Prometheus metrics.
func InitMetrics() *Metrics {

	metrics := &Metrics{}

	metrics.TestedResources = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: PrometheusNamespace,
			Name:      "tested_resources",
			Help:      "The fraction of resources that are currently tested",
		},
		[]string{"type", "status"},
	)

	metrics.Resources = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: PrometheusNamespace,
			Name:      "resources",
			Help:      "The number of resources we have",
		},
		[]string{"type"},
	)

	metrics.Requests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "requests_total",
			Help:      "The number of API requests",
		},
		[]string{"target"},
	)

	return metrics
}
