package database

import (
	"sync"
	"time"

	"server/config"
)

const (
	// PostgreSQL SQLSTATE class prefixes that indicate a code/schema mismatch.
	// These are unrecoverable programming errors and should crash the process.
	sqlStateSyntaxPrefix           = "42"
	sqlStateDataExceptionPrefix    = "22"
	sqlStateInvalidStatementPrefix = "26"
)

var (
	defaultsOnce sync.Once
	defaults     dbDefaults
)

type dbDefaults struct {
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" default:"90"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" default:"25"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" default:"30m"`
}

func loadDefaults() dbDefaults {
	var d dbDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *dbDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// MaxOpenConns returns the maximum number of open database connections.
func MaxOpenConns() int { return getDefaults().MaxOpenConns }

// MaxIdleConns returns the maximum number of idle database connections.
func MaxIdleConns() int { return getDefaults().MaxIdleConns }

// ConnMaxLifetime returns the maximum lifetime of a database connection.
func ConnMaxLifetime() time.Duration { return getDefaults().ConnMaxLifetime }
