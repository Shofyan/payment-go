package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	databaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	databaseConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_idle",
			Help: "Number of idle database connections",
		},
	)
)

// ConnectionPool manages database connections with pooling
type ConnectionPool struct {
	db      *sql.DB
	logger  *zap.Logger
	metrics *PoolMetrics
}

// PoolMetrics tracks connection pool metrics
type PoolMetrics struct {
	mu sync.RWMutex

	activeConnections int
	idleConnections   int
	waitCount         int64
	waitDuration      time.Duration
	maxIdleClosed     int64
	maxLifetimeClosed int64
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	// Connection Pool Settings
	MaxOpenConns    int           // Maximum number of open connections (default: 25)
	MaxIdleConns    int           // Maximum number of idle connections (default: 5)
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection (default: 5 minutes)
	ConnMaxIdleTime time.Duration // Maximum idle time (default: 10 minutes)

	// Timeouts
	ConnectTimeout time.Duration // Connection timeout (default: 10s)
	QueryTimeout   time.Duration // Default query timeout (default: 30s)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            5432,
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		ConnectTimeout:  10 * time.Second,
		QueryTimeout:    30 * time.Second,
	}
}

// NewConnectionPool creates a new database connection pool
func NewConnectionPool(cfg Config, logger *zap.Logger) (*ConnectionPool, error) {
	// Build connection string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
		int(cfg.ConnectTimeout.Seconds()),
	)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &ConnectionPool{
		db:      db,
		logger:  logger,
		metrics: &PoolMetrics{},
	}

	logger.Info("Database connection pool initialized",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Int("max_open_conns", cfg.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.MaxIdleConns),
	)

	// Start metrics collector
	go pool.collectMetrics()

	return pool, nil
}

// GetDB returns the underlying sql.DB
func (cp *ConnectionPool) GetDB() *sql.DB {
	return cp.db
}

// QueryContext executes a query with context
func (cp *ConnectionPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := cp.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		cp.logger.Error("Query failed",
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		return nil, err
	}

	cp.logger.Debug("Query executed",
		zap.Duration("duration", duration),
	)

	return rows, nil
}

// ExecContext executes a query that doesn't return rows
func (cp *ConnectionPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := cp.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		cp.logger.Error("Exec failed",
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		return nil, err
	}

	cp.logger.Debug("Exec completed",
		zap.Duration("duration", duration),
	)

	return result, nil
}

// QueryRowContext executes a query that returns at most one row
func (cp *ConnectionPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return cp.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction
func (cp *ConnectionPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return cp.db.BeginTx(ctx, opts)
}

// Close closes the database connection pool
func (cp *ConnectionPool) Close() error {
	cp.logger.Info("Closing database connection pool")
	return cp.db.Close()
}

// GetStats returns current pool statistics
func (cp *ConnectionPool) GetStats() sql.DBStats {
	return cp.db.Stats()
}

// GetMetrics returns custom metrics
func (cp *ConnectionPool) GetMetrics() PoolMetrics {
	cp.metrics.mu.RLock()
	defer cp.metrics.mu.RUnlock()

	// Return a copy without the mutex
	return PoolMetrics{
		activeConnections: cp.metrics.activeConnections,
		idleConnections:   cp.metrics.idleConnections,
		waitCount:         cp.metrics.waitCount,
		waitDuration:      cp.metrics.waitDuration,
		maxIdleClosed:     cp.metrics.maxIdleClosed,
		maxLifetimeClosed: cp.metrics.maxLifetimeClosed,
	}
}

// IsHealthy checks if database connection is healthy
func (cp *ConnectionPool) IsHealthy(ctx context.Context) bool {
	if err := cp.db.PingContext(ctx); err != nil {
		cp.logger.Warn("Database health check failed", zap.Error(err))
		return false
	}

	// Check if we're running out of connections
	stats := cp.db.Stats()
	utilization := float64(stats.OpenConnections) / float64(stats.MaxOpenConnections)
	if utilization > 0.9 {
		cp.logger.Warn("Database connection pool utilization high",
			zap.Float64("utilization", utilization),
		)
		return false
	}

	return true
}

// collectMetrics periodically collects pool metrics
func (cp *ConnectionPool) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := cp.db.Stats()

		cp.metrics.mu.Lock()
		cp.metrics.activeConnections = stats.InUse
		cp.metrics.idleConnections = stats.Idle
		cp.metrics.waitCount = stats.WaitCount
		cp.metrics.waitDuration = stats.WaitDuration
		cp.metrics.maxIdleClosed = stats.MaxIdleClosed
		cp.metrics.maxLifetimeClosed = stats.MaxLifetimeClosed
		cp.metrics.mu.Unlock()

		// Update Prometheus metrics
		databaseConnectionsActive.Set(float64(stats.InUse))
		databaseConnectionsIdle.Set(float64(stats.Idle))

		// Log if connection pool is stressed
		if stats.WaitCount > 0 {
			cp.logger.Warn("Database connections are waiting",
				zap.Int("in_use", stats.InUse),
				zap.Int("idle", stats.Idle),
				zap.Int64("wait_count", stats.WaitCount),
				zap.Duration("wait_duration", stats.WaitDuration),
			)
		}
	}
}

// WithTimeout wraps a context with query timeout
func (cp *ConnectionPool) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}
