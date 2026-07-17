package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisOpTimeout bounds every coordinator Redis call so a dead/unreachable
// Redis fails a lobby join quickly instead of wedging the HTTP/WS handler.
const redisOpTimeout = 2 * time.Second

// leaseTTL is how long a lobby ownership lease lives without a heartbeat renewal.
const leaseTTL = 15 * time.Second

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

type RedisCoordinator struct {
	rdb *redis.Client
}

// returns a context that gives up after a certain amount of time
func opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), redisOpTimeout)
}

// atomically tries to both claim a lobby and increment epoch number
// keys[1] = lobby owner key, keys[2] = lobby epoch key
// argv[1] = lobby advertise address, argv[2] = ttl
var acquireScript = redis.NewScript(`
local owner = redis.call("GET", KEYS[1])
if owner then
	return {owner, redis.call("GET", KEYS[2]) or "0"}
end
redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2])
return {ARGV[1], redis.call("INCR", KEYS[2])}
`)

func (coord RedisCoordinator) Acquire(name string) (ownerAddr string, epoch int64, err error) {
	ctx, cancel := opCtx()
	defer cancel()
	res, err := acquireScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name)},
		cfg.AdvertiseAddr, int(leaseTTL.Seconds()),
	).Result()
	if err != nil {
		return "", 0, err
	}

	// script returns {ownerAddr, epoch}
	reply, ok := res.([]any)
	if !ok || len(reply) != 2 {
		return "", 0, fmt.Errorf("acquire script: unexpected reply %v", res)
	}
	ownerAddr, ok = reply[0].(string)
	if !ok {
		return "", 0, fmt.Errorf("acquire script: unexpected owner %v", reply[0])
	}
	// INCR (won) yields an integer, GET (lost) yields a string
	switch v := reply[1].(type) {
	case int64:
		epoch = v
	case string:
		epoch, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("acquire script: bad epoch %q: %w", v, err)
		}
	default:
		return "", 0, fmt.Errorf("acquire script: unexpected epoch %v", reply[1])
	}
	return ownerAddr, epoch, nil
}

func (coord RedisCoordinator) Lookup(name string) (ownerAddr string, err error) {
	ctx, cancel := opCtx()
	defer cancel()
	ownerAddr, err = coord.rdb.Get(ctx, lobbyOwnerKey(name)).Result()
	// get returns a nil error if it doesn't exist, but that is a valid non-error case for us
	if err == redis.Nil {
		return "", nil
	}
	return ownerAddr, err
}

// deletes the lobby key if the right lobby owner is requesting it
// keys[1] = lobby owner key, keys[2] = lobby epoch key
// argv[1] = lobby epoch
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

func (coord RedisCoordinator) Release(name string, epoch int64) {
	ctx, cancel := opCtx()
	defer cancel()
	err := releaseScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name)},
		epoch).Err()
	if err != nil {
		log.Printf("release lobby %q: %v (lease will expire on its own)", name, err)
	}
}

// refreshes the key if the right lobby owner is requesting it
// keys[1] = lobby owner key, keys[2] = lobby epoch key
// argv[1] = lobby epoch, argv[2] = ttl
var refreshScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) == ARGV[1] then
	return redis.call("EXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

func (coord RedisCoordinator) Refresh(name string, epoch int64) (bool, error) {
	ctx, cancel := opCtx()
	defer cancel()
	return refreshScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name)},
		epoch, int(leaseTTL.Seconds())).Bool()
}

func lobbyOwnerKey(name string) string {
	return fmt.Sprintf("lobby:%s:owner", name)
}

func lobbyEpochKey(name string) string {
	return fmt.Sprintf("lobby:%s:epoch", name)
}
