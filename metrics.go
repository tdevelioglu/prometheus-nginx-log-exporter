package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// metrics is the struct that contains pointers to our metric containers.
type metrics struct {
	bodyBytes       *prometheus.HistogramVec
	upstreamSeconds *prometheus.HistogramVec
	requestSeconds  *prometheus.HistogramVec
}

// newMetrics creates a new metrics based on the provided application and
// label names.
func newMetrics(application string, labelNames []string) *metrics {
	bodyBytes := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "nginx",
		Name:        "http_body_bytes_sent",
		Help:        "Number of body bytes sent to the client",
		ConstLabels: prometheus.Labels{"application": application},
	}, labelNames)

	requestSeconds := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "nginx",
		Name:        "http_request_time_seconds",
		Help:        "Time spent on processing HTTP requests",
		ConstLabels: prometheus.Labels{"application": application},
	}, labelNames)

	upstreamSeconds := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "nginx",
		Name:        "http_upstream_response_time_seconds",
		Help:        "Time spent on receiving a response from upstream servers",
		ConstLabels: prometheus.Labels{"application": application},
	}, labelNames)

	prometheus.MustRegister(bodyBytes, upstreamSeconds, requestSeconds)

	return &metrics{
		bodyBytes:       bodyBytes,
		requestSeconds:  requestSeconds,
		upstreamSeconds: upstreamSeconds,
	}
}
