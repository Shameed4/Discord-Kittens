package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// prometheus stats
var (
	actionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kittens_actions_total",
		Help: "Player actions successfully applied by lobby event loops on this node.",
	})

	failoversTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kittens_failovers_total",
		Help: "Lobbies adopted from a snapshot after their previous owner was lost.",
	})

	// time how long mutating lobby state takes on Redis to see if async is worth
	snapshotWriteDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "kittens_snapshot_write_seconds",
		Help:    "Time to serialize a lobby and persist it via the coordinator.",
		Buckets: prometheus.DefBuckets,
	})

	// gets number of active lobbies using a live function to avoid drift
	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "kittens_active_lobbies",
		Help: "Lobbies currently owned and running on this node.",
	}, func() float64 {
		lobbiesMutex.Lock()
		defer lobbiesMutex.Unlock()
		return float64(len(lobbies))
	})
)

// wires metrics and health endpoints
func registerMetricsRoutes() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}
