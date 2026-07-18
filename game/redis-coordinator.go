package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type RedisCoordinator struct {
	rdb *redis.Client
}

// returns a context that gives up after a certain amount of time
func opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), redisOpTimeout)
}

// atomically tries to both claim a lobby and increment epoch number
// keys[1] = lobby owner key, keys[2] = lobby epoch key
// argv[1] = node advertise address, argv[2] = ttl
var acquireScript = redis.NewScript(`
local ownerAddr = redis.call("GET", KEYS[1])
if ownerAddr then
	return {ownerAddr, redis.call("GET", KEYS[2]) or "0"}
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
// argv[1] = node advertise address, argv[2] = lobby epoch, argv[3] = ttl
var refreshScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) == ARGV[2] then
	redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[3])
	return 1
end
return 0
`)

func (coord RedisCoordinator) Refresh(name string, epoch int64) (bool, error) {
	ctx, cancel := opCtx()
	defer cancel()
	return refreshScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name)},
		cfg.AdvertiseAddr, epoch, int(leaseTTL.Seconds())).Bool()
}
