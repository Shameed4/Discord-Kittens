package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"
)

func main() {
	target := flag.String("target", "ws://localhost:8080", "WS base URL of the cluster (nginx or a node)")
	numBots := flag.Int("bots", 8, "total number of concurrent bots")
	perLobby := flag.Int("perLobby", 4, "bots grouped into each shared lobby")
	lobbyPrefix := flag.String("lobbyPrefix", "load", "lobby name prefix")
	think := flag.Duration("think", 150*time.Millisecond, "pause before each action, to pace play")
	verbose := flag.Bool("verbose", false, "log every action")
	flag.Parse()

	log.Printf("launching %d bots against %s (%d per lobby)", *numBots, *target, *perLobby)

	var wg sync.WaitGroup
	for i := 0; i < *numBots; i++ {
		b := bot{
			target:    *target,
			lobby:     fmt.Sprintf("%s-%d", *lobbyPrefix, i / *perLobby),
			userId:    fmt.Sprintf("bot-%d", i),
			wantSeats: *perLobby,
			think:     *think,
			verbose:   *verbose,
		}
		wg.Go(b.run)
	}
	wg.Wait()
}
