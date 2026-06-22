package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry            *prometheus.Registry
	notificationCreated *prometheus.CounterVec
	notificationRead    *prometheus.CounterVec
	notificationDeleted *prometheus.CounterVec
	emailJobsQueued     *prometheus.CounterVec
	unreadCount         *prometheus.GaugeVec
	processingDelay     *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		notificationCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "notification_service",
				Name:      "notifications_created_total",
				Help:      "Total notifications created",
			},
			[]string{"category", "channel"},
		),
		notificationRead: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "notification_service",
				Name:      "notifications_read_total",
				Help:      "Total notifications marked as read",
			},
			[]string{"category"},
		),
		notificationDeleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "notification_service",
				Name:      "notifications_deleted_total",
				Help:      "Total notifications deleted",
			},
			[]string{"category"},
		),
		emailJobsQueued: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "dms",
				Subsystem: "notification_service",
				Name:      "email_jobs_queued_total",
				Help:      "Total email scheduler queued",
			},
			[]string{"template_id", "category"},
		),
		unreadCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "dms",
				Subsystem: "notification_service",
				Name:      "unread_notifications",
				Help:      "Unread notifications by category",
			},
			[]string{"category"},
		),
		processingDelay: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "dms",
				Subsystem: "notification_service",
				Name:      "notification_processing_seconds",
				Help:      "Processing delay for notification flows",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"category"},
		),
	}

	registry.MustRegister(
		m.notificationCreated,
		m.notificationRead,
		m.notificationDeleted,
		m.emailJobsQueued,
		m.unreadCount,
		m.processingDelay,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) IncCreated(category, channel string) {
	m.notificationCreated.WithLabelValues(category, channel).Inc()
}

func (m *Metrics) IncRead(category string) {
	m.notificationRead.WithLabelValues(category).Inc()
}

func (m *Metrics) IncDeleted(category string) {
	m.notificationDeleted.WithLabelValues(category).Inc()
}

func (m *Metrics) IncEmailJob(templateID, category string) {
	m.emailJobsQueued.WithLabelValues(templateID, category).Inc()
}

func (m *Metrics) SetUnread(category string, count float64) {
	m.unreadCount.WithLabelValues(category).Set(count)
}

func (m *Metrics) ObserveProcessing(category string, startedAt time.Time) {
	m.processingDelay.WithLabelValues(category).Observe(time.Since(startedAt).Seconds())
}
