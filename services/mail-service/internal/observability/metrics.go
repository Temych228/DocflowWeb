package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry       *prometheus.Registry
	jobsQueued     *prometheus.CounterVec
	jobsProcessing *prometheus.CounterVec
	jobsSent       *prometheus.CounterVec
	jobsFailed     *prometheus.CounterVec
	jobLatency     *prometheus.HistogramVec
	byCategory     *prometheus.CounterVec
	byHour         *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		jobsQueued: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "mail_service",
				Name:      "jobs_queued_total",
				Help:      "Total queued mail jobs",
			},
			[]string{"category", "template_id"},
		),
		jobsProcessing: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "mail_service",
				Name:      "jobs_processing_total",
				Help:      "Total processing mail jobs",
			},
			[]string{"category", "template_id"},
		),
		jobsSent: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "mail_service",
				Name:      "jobs_sent_total",
				Help:      "Total sent mail jobs",
			},
			[]string{"category", "template_id"},
		),
		jobsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "mail_service",
				Name:      "jobs_failed_total",
				Help:      "Total failed mail jobs",
			},
			[]string{"category", "template_id"},
		),
		jobLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "dms",
				Subsystem: "mail_service",
				Name:      "job_processing_seconds",
				Help:      "Mail job processing duration",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"category"},
		),
		byCategory: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "mail_service",
				Name:      "events_by_category_total",
				Help:      "Events by category",
			},
			[]string{"category"},
		),
		byHour: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "mail_service",
				Name:      "events_by_hour_total",
				Help:      "Events by hour",
			},
			[]string{"hour"},
		),
	}

	registry.MustRegister(m.jobsQueued, m.jobsProcessing, m.jobsSent, m.jobsFailed, m.jobLatency, m.byCategory, m.byHour)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) IncQueued(category, templateID string) {
	m.jobsQueued.WithLabelValues(category, templateID).Inc()
}

func (m *Metrics) IncProcessing(category, templateID string) {
	m.jobsProcessing.WithLabelValues(category, templateID).Inc()
}

func (m *Metrics) IncSent(category, templateID string) {
	m.jobsSent.WithLabelValues(category, templateID).Inc()
}

func (m *Metrics) IncFailed(category, templateID string) {
	m.jobsFailed.WithLabelValues(category, templateID).Inc()
}

func (m *Metrics) ObserveLatency(category string, startedAt time.Time) {
	m.jobLatency.WithLabelValues(category).Observe(time.Since(startedAt).Seconds())
}

func (m *Metrics) IncCategory(category string) {
	m.byCategory.WithLabelValues(category).Inc()
}

func (m *Metrics) IncHour(hour string) {
	m.byHour.WithLabelValues(hour).Inc()
}
