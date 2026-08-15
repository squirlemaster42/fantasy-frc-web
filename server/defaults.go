package main

import (
	"server/utils"
	"sync"
	"time"
)

const (
	staticAssetMaxAgeSecondsEnvKey = "STATIC_ASSET_MAX_AGE_SECONDS"
	defaultStaticAssetMaxAgeSeconds = 2592000 // 30 days

	serverShutdownTimeoutEnvKey = "SERVER_SHUTDOWN_TIMEOUT"
	defaultServerShutdownTimeout = 10 * time.Second

	otelServiceName = "fantasy-frc-web"
)

var (
	defaultsOnce sync.Once
	defaults     mainDefaults
)

type mainDefaults struct {
	staticAssetMaxAgeSeconds int
	serverShutdownTimeout    time.Duration
}

func loadDefaults() mainDefaults {
	return mainDefaults{
		staticAssetMaxAgeSeconds: utils.MustGetEnvInt(staticAssetMaxAgeSecondsEnvKey, defaultStaticAssetMaxAgeSeconds),
		serverShutdownTimeout:    utils.MustGetEnvDuration(serverShutdownTimeoutEnvKey, defaultServerShutdownTimeout),
	}
}

func getDefaults() *mainDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// StaticAssetMaxAgeSeconds returns the Cache-Control max-age for static assets.
func StaticAssetMaxAgeSeconds() int { return getDefaults().staticAssetMaxAgeSeconds }

// ServerShutdownTimeout returns the graceful shutdown timeout.
func ServerShutdownTimeout() time.Duration { return getDefaults().serverShutdownTimeout }
