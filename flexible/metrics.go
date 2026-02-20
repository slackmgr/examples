package main

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics implements the Metrics interface using the Prometheus client library.
type PrometheusMetrics struct {
	// A mutex is used to protect the maps from concurrent access.
	mu sync.RWMutex

	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

// NewPrometheusMetrics creates and returns a new PrometheusMetrics instance.
func NewPrometheusMetrics() *PrometheusMetrics {
	m := &PrometheusMetrics{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}

	return m
}

func (m *PrometheusMetrics) RegisterCounter(name, help string, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If the counter is already registered, do nothing.
	if _, ok := m.counters[name]; ok {
		return
	}

	// Create and register the new counter vector.
	// Using promauto will handle registration automatically and safely.
	counterVec := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)

	m.counters[name] = counterVec
}

func (m *PrometheusMetrics) RegisterGauge(name, help string, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If the gauge is already registered, do nothing.
	if _, ok := m.gauges[name]; ok {
		return
	}

	// Create and register the new gauge vector.
	gaugeVec := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labels)

	m.gauges[name] = gaugeVec
}

func (m *PrometheusMetrics) RegisterHistogram(name, help string, buckets []float64, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If the histogram is already registered, do nothing.
	if _, ok := m.histograms[name]; ok {
		return
	}

	// Create and register the new histogram vector.
	histogramVec := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}, labels)

	m.histograms[name] = histogramVec
}

func (m *PrometheusMetrics) Add(name string, value float64, labelValues ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	counterVec, ok := m.counters[name]
	if !ok {
		return
	}

	// Get the specific counter for the given labels and add the value.
	// This is safe for concurrent use.
	counterVec.WithLabelValues(labelValues...).Add(value)
}

func (m *PrometheusMetrics) Inc(name string, labelValues ...string) {
	m.Add(name, 1, labelValues...)
}

func (m *PrometheusMetrics) Set(name string, value float64, labelValues ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gaugeVec, ok := m.gauges[name]
	if !ok {
		return
	}

	// Get the specific gauge for the given labels and set the value.
	// This is safe for concurrent use.
	gaugeVec.WithLabelValues(labelValues...).Set(value)
}

func (m *PrometheusMetrics) Observe(name string, value float64, labelValues ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	histogramVec, ok := m.histograms[name]
	if !ok {
		return
	}

	// Get the specific histogram for the given labels and observe the value.
	// This is safe for concurrent use.
	histogramVec.WithLabelValues(labelValues...).Observe(value)
}
