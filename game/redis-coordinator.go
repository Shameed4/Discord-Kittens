package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisOpTimeout bounds most coordinator Redis calls so a dead/unreachable
// Redis fails a lobby join quickly instead of wedging the HTTP/WS handler.
const redisOpTimeout = 2 * time.Second

// state write is shorter because it happens on every action
const redisStateWriteTimeout = 500 * time.Millisecond

type RedisCoordinator struct {
	rdb *redis.Client
}

// returns a context that gives up after a certain amount of time
func opCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// atomically tries to both claim a lobby and increment epoch number.
// on a win it also returns any snapshot left by a previous (crashed) owner so the
// new owner can resume the game; "" when there's nothing to inherit.
// keys[1] = lobby owner key, keys[2] = lobby epoch key, keys[3] = lobby state key
// argv[1] = node advertise address, argv[2] = ttl
// reply = {ownerAddr, epoch, state}
var acquireScript = redis.NewScript(`
local ownerAddr = redis.call("GET", KEYS[1])
if ownerAddr then
	return {ownerAddr, redis.call("GET", KEYS[2]) or "0", ""}
end
redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2])
return {ARGV[1], redis.call("INCR", KEYS[2]), redis.call("GET", KEYS[3]) or ""}
`)

func (coord RedisCoordinator) Acquire(name string) (ownerAddr string, epoch int64, state []byte, err error) {
	ctx, cancel := opCtx(redisOpTimeout)
	defer cancel()
	res, err := acquireScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name), lobbyStateKey(name)},
		cfg.AdvertiseAddr, int(leaseTTL.Seconds()),
	).Result()
	if err != nil {
		return "", 0, nil, err
	}

	// script returns {ownerAddr, epoch, serialized lobby}
	reply, ok := res.([]any)
	if !ok || len(reply) != 3 {
		return "", 0, nil, fmt.Errorf("acquire script: unexpected reply %v", res)
	}
	ownerAddr, ok = reply[0].(string)
	if !ok {
		return "", 0, nil, fmt.Errorf("acquire script: unexpected owner %v", reply[0])
	}
	// INCR (won) yields an integer, GET (lost) yields a string
	switch v := reply[1].(type) {
	case int64:
		epoch = v
	case string:
		epoch, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return "", 0, nil, fmt.Errorf("acquire script: bad epoch %q: %w", v, err)
		}
	default:
		return "", 0, nil, fmt.Errorf("acquire script: unexpected epoch %v", reply[1])
	}
	// reply[2] is the inherited snapshot ("" when there's nothing to resume)
	if s, ok := reply[2].(string); ok && s != "" {
		state = []byte(s)
	}
	return ownerAddr, epoch, state, nil
}

func (coord RedisCoordinator) Lookup(name string) (ownerAddr string, err error) {
	ctx, cancel := opCtx(redisOpTimeout)
	defer cancel()
	ownerAddr, err = coord.rdb.Get(ctx, lobbyOwnerKey(name)).Result()
	// get returns a nil error if it doesn't exist, but that is a valid non-error case for us
	if err == redis.Nil {
		return "", nil
	}
	return ownerAddr, err
}

// deletes the lobby owner and state keys if the right lobby owner is requesting it.
// dropping the state key marks this as a graceful teardown (reap / game over) so a
// future re-creation of the same lobby name won't wrongly adopt a stale snapshot.
// a crashed owner never runs this, so its state key survives (TTL) for adoption.
// keys[1] = lobby owner key, keys[2] = lobby epoch key, keys[3] = lobby state key
// argv[1] = lobby epoch
var deleteScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) == ARGV[1] then
	return redis.call("DEL", KEYS[1], KEYS[3])
end
return 0
`)

func (coord RedisCoordinator) Delete(name string, epoch int64) {
	ctx, cancel := opCtx(redisOpTimeout)
	defer cancel()
	err := deleteScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name), lobbyStateKey(name)},
		epoch).Err()
	if err != nil {
		log.Printf("delete lobby %q: %v (lease will expire on its own)", name, err)
	}
}

// deletes this lobby's ownership without removing it's state to allow other lobbies
// to gracefully take over. epoch incremented to prevent race with heartbeat
// keys[1] = lobby owner key, keys[2] = lobby epoch key, keys[3] = lobby state key
// argv[1] = lobby epoch
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) == ARGV[1] then
	redis.call("INCR", KEYS[2])
	return redis.call("DEL", KEYS[1])
end
return 0
`)

func (coord RedisCoordinator) Release(name string, epoch int64) {
	ctx, cancel := opCtx(redisOpTimeout)
	defer cancel()
	err := releaseScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name), lobbyStateKey(name)},
		epoch).Err()
	if err != nil {
		log.Printf("release lobby %q: %v (lease will expire on its own)", name, err)
	}
}

// refreshes the key if the right lobby owner is requesting it
// keys[1] = lobby owner key, keys[2] = lobby epoch key, keys[3] = lobby state key
// argv[1] = node advertise address, argv[2] = lobby epoch, argv[3] = owner ttl, argv[4] = state ttl
var refreshScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) == ARGV[2] then
	redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[3])
	redis.call("EXPIRE", KEYS[3], ARGV[4])
	return 1
end
return 0
`)

func (coord RedisCoordinator) Refresh(name string, epoch int64) (bool, error) {
	ctx, cancel := opCtx(redisOpTimeout)
	defer cancel()
	return refreshScript.Run(ctx, coord.rdb,
		[]string{lobbyOwnerKey(name), lobbyEpochKey(name), lobbyStateKey(name)},
		cfg.AdvertiseAddr, epoch, int(leaseTTL.Seconds()), int(stateTTL.Seconds())).Bool()
}

// update the state stored by the lobby
// keys[1] = lobby state key, keys[2] = lobby epoch key
// argv[1] = lobby state, argv[2] = lobby epoch, argv[3] = state ttl
var updateStateScript = redis.NewScript(`
if redis.call("GET", KEYS[2]) == ARGV[2] then
	redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[3])
	return 1
end
return 0
`)

func (coord RedisCoordinator) UpdateState(name string, epoch int64, serializedLobby []byte) {
	ctx, cancel := opCtx(redisStateWriteTimeout)
	defer cancel()
	err := updateStateScript.Run(ctx, coord.rdb,
		[]string{lobbyStateKey(name), lobbyEpochKey(name)},
		serializedLobby, epoch, int(stateTTL.Seconds())).Err()
	if err != nil {
		log.Printf("update lobby %q state: %v", name, err)
	}
}

func lobbyOwnerKey(name string) string {
	return fmt.Sprintf("lobby:%s:owner", name)
}

func lobbyEpochKey(name string) string {
	return fmt.Sprintf("lobby:%s:epoch", name)
}

func lobbyStateKey(name string) string {
	return fmt.Sprintf("lobby:%s:state", name)
}
