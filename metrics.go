package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// metrics is the struct that contains pointers to our metric containers.
type metrics struct {
	bodyBytes             *prometheus.HistogramVec
	upstreamHeaderSeconds *prometheus.HistogramVec
	upstreamSeconds       *prometheus.HistogramVec
	requestSeconds        *prometheus.HistogramVec
}

// newMetrics creates a new metrics based on the provided application and
// label names.
func newMetrics(application string, labelNames []string, histogramBuckets []float64) *metrics {
	bodyBytes := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "nginx",
		Name:        "http_body_bytes_sent",
		Help:        "Number of body bytes sent to the client",
		Buckets:     histogramBuckets,
		ConstLabels: prometheus.Labels{"application": application},
	}, labelNames)

	requestSeconds := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "nginx",
		Name:        "http_request_time_seconds",
		Help:        "Time spent on processing HTTP requests",
		Buckets:     histogramBuckets,
		ConstLabels: prometheus.Labels{"application": application},
	}, labelNames)

	upstreamSeconds := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "nginx",
		Name:        "http_upstream_response_time_seconds",
		Help:        "Time spent on receiving a response from upstream servers",
		Buckets:     histogramBuckets,
		ConstLabels: prometheus.Labels{"application": application},
	}, labelNames)

	upstreamHeaderSeconds := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "nginx",
		Name:        "http_upstream_header_time_seconds",
		Help:        "Time to receiving the first byte of the response header from upstream servers",
		Buckets:     histogramBuckets,
		ConstLabels: prometheus.Labels{"application": application},
	}, labelNames)

	prometheus.MustRegister(bodyBytes, upstreamHeaderSeconds, upstreamSeconds, requestSeconds)

	return &metrics{
		bodyBytes:             bodyBytes,
		requestSeconds:        requestSeconds,
		upstreamSeconds:       upstreamSeconds,
		upstreamHeaderSeconds: upstreamHeaderSeconds,
	}
}
