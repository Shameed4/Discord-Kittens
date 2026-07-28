package main

import (
	"log"
	"sort"
	"time"
)

// botMetrics is written by a single bot's goroutine (so no locking), then
// merged across all bots once the run stops.
type botMetrics struct {
	connects   int             // successful dials (initial + every reconnect)
	games      int             // completed games (counted once per lobby, by seat 0)
	actions    int             // actions written to the server
	latencies  []time.Duration // per-action send -> ack round trip
	started    int             // games that reached in-progress
	orphaned   int             // in-progress games that went silent and never came back
	stalled    int             // games that went quiet past orphanTimeout but did come back
	recoveries []time.Duration // drop -> first state back, one per bot per outage
}

// report aggregates every bot's metrics and prints a summary.
func report(mets []botMetrics, elapsed time.Duration, numBots, perLobby int) {
	var connects, games, actions, started, orphaned, stalled int
	var lat, rec []time.Duration
	for i := range mets {
		connects += mets[i].connects
		games += mets[i].games
		actions += mets[i].actions
		started += mets[i].started
		orphaned += mets[i].orphaned
		stalled += mets[i].stalled
		lat = append(lat, mets[i].latencies...)
		rec = append(rec, mets[i].recoveries...)
	}
	// every dial past the first per bot is a reconnect
	reconnects := max(0, connects-numBots)
	rate := float64(actions) / elapsed.Seconds()

	log.Printf("=== load test report ===")
	log.Printf("duration:       %s", elapsed.Round(time.Millisecond))
	log.Printf("bots:           %d (%d per lobby)", numBots, perLobby)
	log.Printf("connections:    %d (reconnects: %d)", connects, reconnects)
	log.Printf("games finished: %d", games)
	// games survived: of the games that reached play, how many were never lost
	// to a node death (an orphaned game went silent mid-flight and never
	// recovered on a surviving node). the chaos-run target is 100%.
	survived := started - orphaned
	survivalPct := 100.0
	if started > 0 {
		survivalPct = float64(survived) / float64(started) * 100
	}
	log.Printf("games survived: %d/%d (%.1f%%)", survived, started, survivalPct)
	if stalled > 0 {
		log.Printf("                (%d came back only after %s+ of silence)", stalled, orphanTimeout)
	}
	log.Printf("actions:        %d (%.1f/s)", actions, rate)
	// how long a client waits between losing its node and hearing from the game
	// again — the number the chaos run is really trying to pin down.
	if len(rec) > 0 {
		sort.Slice(rec, func(i, j int) bool { return rec[i] < rec[j] })
		log.Printf("recovery:       p50=%s  p99=%s  max=%s  (%d drops)",
			pctl(rec, 50).Round(time.Millisecond),
			pctl(rec, 99).Round(time.Millisecond),
			rec[len(rec)-1].Round(time.Millisecond), len(rec))
	}
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
