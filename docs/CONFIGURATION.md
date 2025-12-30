# Environment Variables Configuration

This document describes all available environment variables for configuring the Payment API.

## 📋 Table of Contents

- [Server Configuration](#server-configuration)
- [Database Configuration](#database-configuration)
- [Redis Configuration](#redis-configuration)
- [Worker Pool Configuration](#worker-pool-configuration)
- [Rate Limiting Configuration](#rate-limiting-configuration)
- [Circuit Breaker Configuration](#circuit-breaker-configuration)
- [Logging Configuration](#logging-configuration)
- [Monitoring Configuration](#monitoring-configuration)
- [General Configuration](#general-configuration)

## Server Configuration

### `SERVER_ADDRESS`
- **Description**: Address and port for the HTTP server to bind to
- **Default**: `:8080`
- **Example**: `:8080`, `0.0.0.0:3000`
- **Required**: Yes

### `SERVER_READ_TIMEOUT`
- **Description**: Maximum duration for reading the entire request, including body
- **Default**: `15s`
- **Format**: Duration string (e.g., `30s`, `1m`, `500ms`)
- **Required**: No

### `SERVER_WRITE_TIMEOUT`
- **Description**: Maximum duration before timing out writes of the response
- **Default**: `15s`
- **Format**: Duration string
- **Required**: No

### `SERVER_IDLE_TIMEOUT`
- **Description**: Maximum time to wait for the next request when keep-alives are enabled
- **Default**: `60s`
- **Format**: Duration string
- **Required**: No

## Database Configuration

### `DB_HOST`
- **Description**: PostgreSQL database host
- **Default**: `localhost`
- **Example**: `postgres`, `db.example.com`
- **Required**: Yes

### `DB_PORT`
- **Description**: PostgreSQL database port
- **Default**: `5432`
- **Example**: `5432`, `5433`
- **Required**: No

### `DB_USER`
- **Description**: Database username
- **Default**: `postgres`
- **Required**: Yes

### `DB_PASSWORD`
- **Description**: Database password
- **Default**: `postgres`
- **Security**: Keep this secret! Use secrets management in production
- **Required**: Yes

### `DB_NAME`
- **Description**: Database name
- **Default**: `payment_db`
- **Required**: Yes

### `DB_SSLMODE`
- **Description**: SSL mode for database connection
- **Default**: `disable`
- **Options**: `disable`, `require`, `verify-ca`, `verify-full`
- **Required**: No
- **Note**: Use `require` or higher in production

### `DB_MAX_OPEN_CONNS`
- **Description**: Maximum number of open connections to the database
- **Default**: `25`
- **Range**: 1-100 (depends on database capacity)
- **Required**: No
- **Tuning**: Set based on expected load and database resources

### `DB_MAX_IDLE_CONNS`
- **Description**: Maximum number of idle connections in the pool
- **Default**: `5`
- **Range**: 1-20
- **Required**: No
- **Note**: Should be less than `DB_MAX_OPEN_CONNS`

### `DB_CONN_MAX_LIFETIME`
- **Description**: Maximum lifetime of a database connection
- **Default**: `5m`
- **Format**: Duration string
- **Required**: No
- **Note**: Helps recycle connections and prevent stale connections

### `DB_CONN_MAX_IDLE_TIME`
- **Description**: Maximum time a connection can be idle before being closed
- **Default**: `10m`
- **Format**: Duration string
- **Required**: No

### `DB_CONNECT_TIMEOUT`
- **Description**: Timeout for establishing database connection
- **Default**: `10s`
- **Format**: Duration string
- **Required**: No

### `DB_QUERY_TIMEOUT`
- **Description**: Default timeout for database queries
- **Default**: `30s`
- **Format**: Duration string
- **Required**: No

## Redis Configuration

### `REDIS_HOST`
- **Description**: Redis server host
- **Default**: `localhost`
- **Example**: `redis`, `cache.example.com`
- **Required**: No (if not using Redis features)

### `REDIS_PORT`
- **Description**: Redis server port
- **Default**: `6379`
- **Required**: No

### `REDIS_PASSWORD`
- **Description**: Redis authentication password
- **Default**: `` (empty, no auth)
- **Security**: Keep this secret!
- **Required**: No

### `REDIS_DB`
- **Description**: Redis database number
- **Default**: `0`
- **Range**: 0-15
- **Required**: No

## Worker Pool Configuration

### `WORKER_COUNT`
- **Description**: Number of concurrent worker goroutines
- **Default**: `50`
- **Range**: 10-500
- **Required**: No
- **Tuning**: 
  - Higher = more concurrent processing
  - Too high = more memory and CPU overhead
  - Recommended: 50-100 for most workloads

### `QUEUE_SIZE`
- **Description**: Size of the task queue (backpressure threshold)
- **Default**: `1000`
- **Range**: 100-10000
- **Required**: No
- **Tuning**: 
  - When queue is full, new requests get 503
  - Higher = more buffering before backpressure
  - Lower = faster rejection of excess load

### `WORKER_SHUTDOWN_TIMEOUT`
- **Description**: Maximum time to wait for workers to finish during graceful shutdown
- **Default**: `30s`
- **Format**: Duration string
- **Required**: No

## Rate Limiting Configuration

### `RATE_LIMIT_REQUESTS_PER_SECOND`
- **Description**: Number of requests allowed per second per user/IP
- **Default**: `100`
- **Type**: Float
- **Range**: 1.0-10000.0
- **Required**: No
- **Example**: `100`, `250.5`, `1000`

### `RATE_LIMIT_BURST_SIZE`
- **Description**: Maximum burst size (token bucket capacity)
- **Default**: `200`
- **Range**: Equal to or greater than requests per second
- **Required**: No
- **Note**: Allows temporary bursts above the sustained rate

## Circuit Breaker Configuration

### `CIRCUIT_BREAKER_MAX_FAILURES`
- **Description**: Number of consecutive failures before opening the circuit
- **Default**: `5`
- **Range**: 3-20
- **Required**: No
- **Tuning**: Lower = more sensitive to failures

### `CIRCUIT_BREAKER_TIMEOUT`
- **Description**: Duration to wait in open state before entering half-open
- **Default**: `60s`
- **Format**: Duration string
- **Required**: No
- **Note**: Time to wait before testing if service recovered

### `CIRCUIT_BREAKER_HALF_OPEN_SUCCESS`
- **Description**: Number of successful requests needed to close from half-open
- **Default**: `2`
- **Range**: 1-10
- **Required**: No

## Logging Configuration

### `LOG_LEVEL`
- **Description**: Minimum log level
- **Default**: `info`
- **Options**: `debug`, `info`, `warn`, `error`, `fatal`
- **Required**: No
- **Production**: Use `info` or `warn`
- **Development**: Use `debug`

### `LOG_FORMAT`
- **Description**: Log output format
- **Default**: `json`
- **Options**: `json`, `console`
- **Required**: No
- **Production**: Use `json` for structured logging
- **Development**: Use `console` for readability

## Monitoring Configuration

### `PROMETHEUS_ENABLED`
- **Description**: Enable Prometheus metrics endpoint
- **Default**: `true`
- **Type**: Boolean
- **Required**: No

### `METRICS_PATH`
- **Description**: HTTP path for Prometheus metrics
- **Default**: `/metrics`
- **Required**: No

## General Configuration

### `ENV`
- **Description**: Environment name
- **Default**: `development`
- **Options**: `development`, `staging`, `production`
- **Required**: No
- **Note**: Affects default behaviors (logging, error details, etc.)

## 🚀 Usage Examples

### Local Development

Create `.env` file:
```bash
# Copy example
cp .env.example .env

# Edit with your local settings
SERVER_ADDRESS=:8080
DB_HOST=localhost
DB_PASSWORD=mypassword
ENV=development
LOG_LEVEL=debug
LOG_FORMAT=console
```

Run application:
```bash
# Load .env automatically (if using godotenv)
go run cmd/api/main.go

# Or export manually
export $(cat .env | xargs)
go run cmd/api/main.go
```

### Docker Compose

```yaml
services:
  api:
    build: .
    env_file:
      - .env
    environment:
      - DB_HOST=postgres  # Override specific variables
      - REDIS_HOST=redis
```

### Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: payment-api-config
data:
  SERVER_ADDRESS: ":8080"
  DB_HOST: "postgres-service"
  ENV: "production"
  LOG_LEVEL: "info"
---
apiVersion: v1
kind: Secret
metadata:
  name: payment-api-secrets
type: Opaque
stringData:
  DB_PASSWORD: "your-secret-password"
  REDIS_PASSWORD: "redis-secret"
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: api
        envFrom:
        - configMapRef:
            name: payment-api-config
        - secretRef:
            name: payment-api-secrets
```

### Production Environment Variables

```bash
# Server
export SERVER_ADDRESS=":8080"
export SERVER_READ_TIMEOUT="30s"
export SERVER_WRITE_TIMEOUT="30s"

# Database (use secrets management!)
export DB_HOST="prod-db.example.com"
export DB_PORT="5432"
export DB_USER="payment_user"
export DB_PASSWORD="${DB_PASSWORD_FROM_SECRETS}"
export DB_NAME="payment_production"
export DB_SSLMODE="require"
export DB_MAX_OPEN_CONNS="50"

# Performance tuning
export WORKER_COUNT="100"
export QUEUE_SIZE="2000"
export RATE_LIMIT_REQUESTS_PER_SECOND="200"

# Production settings
export ENV="production"
export LOG_LEVEL="info"
export LOG_FORMAT="json"
```

## 🔒 Security Best Practices

1. **Never commit `.env` to git** - Use `.env.example` instead
2. **Use secrets management** in production:
   - Kubernetes Secrets
   - AWS Secrets Manager
   - HashiCorp Vault
   - Azure Key Vault
3. **Rotate credentials regularly**
4. **Use SSL/TLS** for database connections (`DB_SSLMODE=require`)
5. **Restrict access** to environment variables
6. **Audit configuration changes**

## 🧪 Testing Different Configurations

### Test with environment variables:
```bash
WORKER_COUNT=10 QUEUE_SIZE=100 go run cmd/api/main.go
```

### Test with .env file:
```bash
cp .env.example .env.test
# Edit .env.test
ENV_FILE=.env.test go run cmd/api/main.go
```

## 📊 Performance Tuning Guide

| Workload | Worker Count | Queue Size | DB Connections | Rate Limit |
|----------|--------------|------------|----------------|------------|
| **Light** (< 1K req/s) | 20-50 | 500 | 10-20 | 50-100 |
| **Medium** (1K-5K req/s) | 50-100 | 1000 | 20-30 | 100-200 |
| **Heavy** (5K-10K req/s) | 100-200 | 2000 | 30-50 | 200-500 |
| **Very Heavy** (> 10K req/s) | 200-500 | 5000 | 50-100 | 500-1000 |

**Note**: These are starting points. Monitor and adjust based on actual metrics.

## 🔍 Troubleshooting

### Issue: Application won't start

**Check**: Required environment variables are set
```bash
echo $DB_HOST
echo $DB_PASSWORD
```

### Issue: Database connection errors

**Check**: Database configuration and connectivity
```bash
# Test database connection
psql -h $DB_HOST -U $DB_USER -d $DB_NAME
```

### Issue: High memory usage

**Adjust**: Reduce worker count and queue size
```bash
export WORKER_COUNT=25
export QUEUE_SIZE=500
```

### Issue: Frequent 503 errors

**Adjust**: Increase worker count or queue size
```bash
export WORKER_COUNT=100
export QUEUE_SIZE=2000
```

### Issue: Database connection pool exhausted

**Adjust**: Increase database connections
```bash
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10
```

---

**For more information, see:**
- [README.md](../README.md)
- [ARCHITECTURE.md](../ARCHITECTURE.md)
