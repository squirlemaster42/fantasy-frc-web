// Package redisutil provides shared helpers for creating Redis clients used
// throughout the application.
package redisutil

import "github.com/redis/go-redis/v9"

const protocolVersion = 2

// NewClient creates a go-redis client with the application's standard settings.
// The caller is responsible for verifying connectivity (e.g., Ping) and closing
// the client when done.
func NewClient(addr string, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		Protocol: protocolVersion,
	})
}
