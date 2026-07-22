package main

import (
	"log"
	"sort"
	"time"
)

// botMetrics is written by a single bot's goroutine (so no locking), then
// merged across all bots once the run stops.
type botMetrics struct {
	connects  int             // successful dials (initial + every reconnect)
	games     int             // completed games (counted once per lobby, by seat 0)
	actions   int             // actions written to the server
	latencies []time.Duration // per-action send -> ack round trip
}

// report aggregates every bot's metrics and prints a summary.
func report(mets []botMetrics, elapsed time.Duration, numBots, perLobby int) {
	var connects, games, actions int
	var lat []time.Duration
	for i := range mets {
		connects += mets[i].connects
		games += mets[i].games
		actions += mets[i].actions
		lat = append(lat, mets[i].latencies...)
	}
	// every dial past the first per bot is a reconnect
	reconnects := max(0, connects-numBots)
	rate := float64(actions) / elapsed.Seconds()

	log.Printf("=== load test report ===")
	log.Printf("duration:       %s", elapsed.Round(time.Millisecond))
	log.Printf("bots:           %d (%d per lobby)", numBots, perLobby)
	log.Printf("connections:    %d (reconnects: %d)", connects, reconnects)
	log.Printf("games finished: %d", games)
	log.Printf("actions:        %d (%.1f/s)", actions, rate)
	if len(lat) > 0 {
		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		log.Printf("action latency: p50=%s  p99=%s  max=%s",
			pctl(lat, 50).Round(time.Microsecond),
			pctl(lat, 99).Round(time.Microsecond),
			lat[len(lat)-1].Round(time.Microsecond))
	}
}

// pctl returns the p-th percentile of an already-sorted slice (nearest-rank).
func pctl(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[p*(len(sorted)-1)/100]
}
