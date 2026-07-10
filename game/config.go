package main

import "os"

type Config struct {
	Port          string
	NodeID        string
	AdvertiseAddr string
	RedisURL      string
}

func LoadConfig() Config {
	host, err := os.Hostname()
	if err != nil {
		host = "node"
	}
	port := getenv("PORT", "8080")
	return Config{
		Port:          port,
		NodeID:        getenv("NODE_ID", host),
		AdvertiseAddr: getenv("ADVERTISE_ADDR", host+":"+port),
		RedisURL:      os.Getenv("REDIS_URL"),
	}
}

func getenv(key string, defaultValue string) string {
	val := os.Getenv(key)
	if len(val) > 0 {
		return val
	}
	return defaultValue
}
