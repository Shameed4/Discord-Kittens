package main

type Coordinator interface {
	// tries to take ownership of a lobby, returning the address if it already exists
	// in another node. it will return its own address if this node owns it.
	Acquire(name string) (ownerAddr string, err error)
	// finds which node owns this lobby without trying to claim it. it will return
	// its own address if this node owns it, or "" if not found.
	Lookup(name string) (ownerAddr string, err error)
	// deletes the lobby so other lobbies know it doesn't exist
	Release(name string)
}

// Local coordinator methods are mostly no-ops to ensure compatability with the RedisCoordinator
type LocalCoordinator struct{}

func (coord LocalCoordinator) Acquire(name string) (ownerAddr string, err error) {
	return cfg.AdvertiseAddr, nil
}

func (coord LocalCoordinator) Lookup(name string) (ownerAddr string, err error) {
	return "", nil
}

// intentional no op
func (coord LocalCoordinator) Release(name string) {}
