package main

import (
	"sync"
	"time"

	"server/config"
)

const otelServiceName = "fantasy-frc-web"

var (
	defaultsOnce sync.Once
	defaults     mainDefaults
)

type mainDefaults struct {
	StaticAssetMaxAgeSeconds int           `env:"STATIC_ASSET_MAX_AGE_SECONDS" default:"2592000"`
	ServerShutdownTimeout    time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" default:"10s"`
	ServerReadHeaderTimeout  time.Duration `env:"SERVER_READ_HEADER_TIMEOUT" default:"60s"`
}

func loadDefaults() mainDefaults {
	var d mainDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *mainDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// StaticAssetMaxAgeSeconds returns the Cache-Control max-age for static assets.
func StaticAssetMaxAgeSeconds() int { return getDefaults().StaticAssetMaxAgeSeconds }

// ServerShutdownTimeout returns the graceful shutdown timeout.
func ServerShutdownTimeout() time.Duration { return getDefaults().ServerShutdownTimeout }

// ServerReadHeaderTimeout returns the HTTP server read header timeout.
func ServerReadHeaderTimeout() time.Duration { return getDefaults().ServerReadHeaderTimeout }
