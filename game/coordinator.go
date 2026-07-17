package main

import (
	"log"

	"github.com/redis/go-redis/v9"
)

var coordinator Coordinator

type Coordinator interface {
	// tries to take ownership of a lobby, returning the address if it already exists
	// in another node. it will return its own address if this node owns it.
	Acquire(name string) (ownerAddr string, err error)
	// finds which node owns this lobby without trying to claim it. it will return
	// its own address if this node owns it, or "" if not found.
	Lookup(name string) (ownerAddr string, err error)
	// deletes the lobby so other nodes know it doesn't exist
	Release(name string)
}

// Local coordinator methods are mostly no-ops to ensure compatability with the RedisCoordinator
type LocalCoordinator struct{}

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

func (coord LocalCoordinator) Acquire(name string) (ownerAddr string, err error) {
	return cfg.AdvertiseAddr, nil
}

func (coord LocalCoordinator) Lookup(name string) (ownerAddr string, err error) {
	return "", nil
}

// intentional no op
func (coord LocalCoordinator) Release(name string) {}

type RedisCoordinator struct {
	rdb *redis.Client
}

func (coord RedisCoordinator) Acquire(name string) (ownerAddr string, err error) {
	return cfg.AdvertiseAddr, nil
}

func (coord RedisCoordinator) Lookup(name string) (ownerAddr string, err error) {
	return "", nil
}

func (coord RedisCoordinator) Release(name string) {}
