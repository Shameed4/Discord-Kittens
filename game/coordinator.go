package main

import (
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisOpTimeout bounds every coordinator Redis call so a dead/unreachable
// Redis fails a lobby join quickly instead of wedging the HTTP/WS handler.
const redisOpTimeout = 2 * time.Second

// leaseTTL is how long a lobby ownership lease lives without a heartbeat renewal.
const leaseTTL = 15 * time.Second
const heartbeatInterval = leaseTTL / 3

var coordinator Coordinator

type Coordinator interface {
	// tries to take ownership of a lobby, returning the address if it already exists
	// in another node. it will return its own address if this node owns it.
	// epoch incremented as a fencing token (the true owner will have the highest epoch)
	Acquire(name string) (ownerAddr string, epoch int64, err error)
	// finds which node owns this lobby without trying to claim it. it will return
	// its own address if this node owns it, or "" if not found.
	Lookup(name string) (ownerAddr string, err error)
	// deletes the lobby so other nodes know it doesn't exist
	Release(name string, epoch int64)
	// refreshes lobby ttl
	Refresh(name string, epoch int64) (success bool, err error)
}

func newCoordinator() Coordinator {
	if cfg.RedisURL == "" {
		return LocalCoordinator{}
	}
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Bad REDIS_URL: %v", err)
	}
	return RedisCoordinator{
		rdb: redis.NewClient(opts),
	}
}

// Local coordinator methods are mostly no-ops to ensure compatability with the RedisCoordinator
type LocalCoordinator struct{}

func (coord LocalCoordinator) Acquire(name string) (ownerAddr string, epoch int64, err error) {
	return cfg.AdvertiseAddr, 1, nil
}

func (coord LocalCoordinator) Lookup(name string) (ownerAddr string, err error) {
	return "", nil
}

// intentional no ops
func (coord LocalCoordinator) Release(name string, epoch int64) {}

func (coord LocalCoordinator) Refresh(name string, epoch int64) (bool, error) {
	return true, nil
}

func lobbyOwnerKey(name string) string {
	return fmt.Sprintf("lobby:%s:owner", name)
}

func lobbyEpochKey(name string) string {
	return fmt.Sprintf("lobby:%s:epoch", name)
}
