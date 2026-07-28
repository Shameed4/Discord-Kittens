package main

import (
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// leaseTTL is how long a lobby ownership lease lives without a heartbeat renewal.
const leaseTTL = 15 * time.Second
const heartbeatInterval = leaseTTL / 3
const stateTTL = 10 * time.Minute

var coordinator Coordinator

type Coordinator interface {
	// tries to take ownership of a lobby, returning the address if it already exists
	// in another node. it will return its own address if this node owns it.
	// epoch incremented as a fencing token (the true owner will have the highest epoch).
	// on a win, state holds any snapshot left by a previous (crashed) owner so the new
	// owner can resume the in-progress game; empty when there's nothing to inherit.
	Acquire(name string) (ownerAddr string, epoch int64, state []byte, err error)
	// finds which node owns this lobby without trying to claim it. it will return
	// its own address if this node owns it, or "" if not found.
	Lookup(name string) (ownerAddr string, err error)
	// deletes the lobby so other nodes know it doesn't exist
	Delete(name string, epoch int64)
	// removes server lease without deleting lobby
	Release(name string, epoch int64)
	// refreshes lobby ttl
	Refresh(name string, epoch int64) (success bool, err error)
	// updates state of lobby
	UpdateState(name string, epoch int64, serializedLobby []byte)
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

func (coord LocalCoordinator) Acquire(name string) (ownerAddr string, epoch int64, state []byte, err error) {
	return cfg.AdvertiseAddr, 1, nil, nil
}

func (coord LocalCoordinator) Lookup(name string) (ownerAddr string, err error) {
	return "", nil
}

func (coord LocalCoordinator) Refresh(name string, epoch int64) (bool, error) {
	return true, nil
}

// intentional no ops
func (coord LocalCoordinator) Delete(name string, epoch int64)                              {}
func (coord LocalCoordinator) Release(name string, epoch int64)                             {}
func (coord LocalCoordinator) UpdateState(name string, epoch int64, serializedLobby []byte) {}
