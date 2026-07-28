package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	target := flag.String("target", "ws://localhost:8080", "WS base URL of the cluster (nginx or a node)")
	numBots := flag.Int("bots", 8, "total number of concurrent bots")
	perLobby := flag.Int("perLobby", 4, "bots grouped into each shared lobby")
	lobbyPrefix := flag.String("lobbyPrefix", "load", "lobby name prefix")
	think := flag.Duration("think", 150*time.Millisecond, "pause before each action, to pace play")
	playChance := flag.Float64("playChance", 0.5, "odds [0-1] a bot plays a card instead of just drawing on its turn")
	nopeChance := flag.Float64("nopeChance", 0.15, "odds [0-1] a bot nopes a pending action when holding a Nope")
	duration := flag.Duration("duration", 0, "run for this long then stop cleanly (0 = run until Ctrl+C)")
	restart := flag.Bool("restart", true, "restart each lobby after game over to sustain load")
	verbose := flag.Bool("verbose", false, "log every action")
	flag.Parse()

	// Ctrl+C / SIGTERM, or the optional -duration, ends the run gracefully:
	// bots stop reconnecting, close their sockets, and return.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

	log.Printf("launching %d bots against %s (%d per lobby)", *numBots, *target, *perLobby)
	start := time.Now()

	// one metrics struct per bot, each written only by its own goroutine
	mets := make([]botMetrics, *numBots)
	var wg sync.WaitGroup
	for i := 0; i < *numBots; i++ {
		b := bot{
			target:     *target,
			lobby:      fmt.Sprintf("%s-%d", *lobbyPrefix, i / *perLobby),
			userId:     fmt.Sprintf("bot-%d", i),
			wantSeats:  *perLobby,
			think:      *think,
			playChance: *playChance,
			nopeChance: *nopeChance,
			restart:    *restart,
			verbose:    *verbose,
			m:          &mets[i],
		}
		wg.Go(func() { b.run(ctx) })
	}
	wg.Wait()
	report(mets, time.Since(start), *numBots, *perLobby)
}
