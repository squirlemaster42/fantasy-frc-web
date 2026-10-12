package database

import (
	"server/utils"
	"sync"
	"time"
)

const (
	maxOpenConnsEnvKey    = "DB_MAX_OPEN_CONNS"
	defaultMaxOpenConns   = 90
	maxIdleConnsEnvKey    = "DB_MAX_IDLE_CONNS"
	defaultMaxIdleConns   = 25
	connMaxLifetimeEnvKey = "DB_CONN_MAX_LIFETIME"
	defaultConnMaxLifetime = 30 * time.Minute

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
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
}

func loadDefaults() dbDefaults {
	return dbDefaults{
		maxOpenConns:    utils.MustGetEnvInt(maxOpenConnsEnvKey, defaultMaxOpenConns),
		maxIdleConns:    utils.MustGetEnvInt(maxIdleConnsEnvKey, defaultMaxIdleConns),
		connMaxLifetime: utils.MustGetEnvDuration(connMaxLifetimeEnvKey, defaultConnMaxLifetime),
	}
}

func getDefaults() *dbDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// MaxOpenConns returns the maximum number of open database connections.
func MaxOpenConns() int { return getDefaults().maxOpenConns }

// MaxIdleConns returns the maximum number of idle database connections.
func MaxIdleConns() int { return getDefaults().maxIdleConns }

// ConnMaxLifetime returns the maximum lifetime of a database connection.
func ConnMaxLifetime() time.Duration { return getDefaults().connMaxLifetime }
