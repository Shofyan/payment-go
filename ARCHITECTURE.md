# High-Throughput Payment API System Architecture

## 🏗️ System Architecture Overview

This is a production-ready, high-throughput payment processing system built with **Go**, implementing **Clean Architecture**, **Domain-Driven Design (DDD)**, and **Docker** for containerization.

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Load Balancer (NGINX)                    │
│              - Least Connection Algorithm                    │
│              - Rate Limiting (100 req/s)                    │
│              - Connection Pooling                            │
└──────────────┬──────────────┬──────────────┬────────────────┘
               │              │              │
       ┌───────▼──────┐ ┌────▼──────┐ ┌────▼──────┐
       │   API-1      │ │   API-2    │ │   API-3    │
       │ (Stateless)  │ │ (Stateless)│ │ (Stateless)│
       └───────┬──────┘ └────┬──────┘ └────┬──────┘
               │              │              │
       ┌───────▼──────────────▼──────────────▼────────┐
       │         Shared Infrastructure                  │
       │  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
       │  │PostgreSQL│  │  Redis   │  │Prometheus │  │
       │  └──────────┘  └──────────┘  └───────────┘  │
       └───────────────────────────────────────────────┘
```

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                 Interface Layer                          │
│  ├─ HTTP Handlers (Controllers)                         │
│  ├─ Middleware (Auth, Rate Limit, Logging)             │
│  └─ Router (Chi)                                         │
└──────────────────┬──────────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────────┐
│              Application Layer                           │
│  ├─ Use Cases (Business Workflows)                      │
│  └─ DTOs (Request/Response)                             │
└──────────────────┬──────────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────────┐
│                Domain Layer                              │
│  ├─ Aggregates (Payment)                                │
│  ├─ Value Objects (Money, Status)                       │
│  ├─ Domain Events                                        │
│  ├─ Repository Interfaces                                │
│  └─ Business Rules                                       │
└──────────────────┬──────────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────────┐
│            Infrastructure Layer                          │
│  ├─ Repository Implementations (PostgreSQL)             │
│  ├─ Payment Gateway (External Services)                 │
│  ├─ Worker Pool                                          │
│  ├─ Rate Limiter                                         │
│  ├─ Circuit Breaker                                      │
│  └─ Connection Pool                                      │
└─────────────────────────────────────────────────────────┘
```

## 🔄 Request Flow (Step-by-Step)

### Normal Payment Processing Flow

```
1. Client Request
   └─> NGINX Load Balancer
       ├─ Rate Limiting Check (100 req/s per IP)
       ├─ Connection Limit Check (10 conn/IP)
       └─> Round-robin to healthy backend

2. API Instance Receives Request
   ├─> Recovery Middleware (panic protection)
   ├─> Logging Middleware (request logging)
   ├─> Timeout Middleware (30s timeout)
   ├─> Backpressure Middleware (health check)
   │   └─ Checks: Worker pool < 90% full
   │              Database connections healthy
   └─> Rate Limit Middleware (token bucket)

3. Payment Handler
   ├─> Parse & Validate Request
   ├─> Create Payment Domain Aggregate
   └─> Save Payment (Status: PENDING)

4. Worker Pool Submission
   ├─> Check Queue Capacity (1000 tasks)
   │   ├─ If Full: Return 503 (Backpressure)
   │   └─ If Available: Submit task
   └─> Return 202 Accepted (Async processing)

5. Worker Goroutine Processes Task
   ├─> Create 30s timeout context
   ├─> Circuit Breaker Check
   │   ├─ CLOSED: Allow request
   │   ├─ OPEN: Return cached/fallback
   │   └─ HALF-OPEN: Allow test request
   │
   ├─> Call Payment Gateway
   │   ├─ With timeout (30s)
   │   └─ With circuit breaker protection
   │
   ├─> Update Payment Status
   │   ├─ Success: PROCESSING -> COMPLETED
   │   └─ Failure: PROCESSING -> FAILED
   │
   └─> Database Update (with optimistic locking)

6. Response to Client
   ├─> 202 Accepted (immediate)
   └─> Client polls GET /payments/{id} for status
```

## 🔑 Key Components and Responsibilities

### 1. Load Balancing (NGINX)

**Purpose**: Distribute traffic across multiple API instances

**Implementation**:
```nginx
upstream payment_api {
    least_conn;  # Route to server with least connections
    server api-1:8080 max_fails=3 fail_timeout=30s;
    server api-2:8080 max_fails=3 fail_timeout=30s;
    server api-3:8080 max_fails=3 fail_timeout=30s;
    keepalive 32;  # Reuse connections
}
```

**Benefits**:
- **Stateless Services**: No session affinity needed
- **Health Checks**: Automatic failover if instance fails
- **Connection Reuse**: Keepalive reduces latency
- **Least Connection**: Routes to least busy server

### 2. Horizontal Scaling

**Auto-scaling Strategy**:
```yaml
# docker-compose.yml (can be replaced with Kubernetes HPA)
services:
  api:
    deploy:
      replicas: 3  # Start with 3 instances
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
```

**Scaling Triggers**:
- CPU > 70%
- Worker pool queue > 80% full
- Response time > 500ms
- Request rate > 300 req/s per instance

**Stateless Design**:
- No in-memory session storage
- Shared database for persistence
- Redis for distributed rate limiting
- Payment status tracked in database

### 3. Worker Pool Pattern

**Implementation** ([pool.go](internal/infrastructure/workerpool/pool.go)):

```go
type WorkerPool struct {
    workerCount int           // 50 workers
    taskQueue   chan Task     // 1000 task buffer (BACKPRESSURE)
    wg          sync.WaitGroup
}

// Submit task - BACKPRESSURE applied here
func (wp *WorkerPool) Submit(ctx context.Context, task Task) error {
    select {
    case wp.taskQueue <- task:
        return nil  // Task queued
    default:
        return ErrQueueFull  // BACKPRESSURE: reject request
    }
}
```

**Key Features**:
- **Fixed Worker Pool**: Prevents goroutine explosion
- **Bounded Queue**: Provides backpressure when overloaded
- **Graceful Shutdown**: Waits for in-flight tasks (30s)
- **Metrics**: Track tasks processed, failed, average duration

### 4. Rate Limiting

**Token Bucket Algorithm** ([limiter.go](internal/infrastructure/ratelimiter/limiter.go)):

```go
// 100 requests per second, burst up to 200
limiter := NewTokenBucketLimiter(100.0, 200)

// Per-user rate limiting
allowed, _ := limiter.Allow(ctx, userID)
if !allowed {
    return 429  // Too Many Requests
}
```

**Three Algorithms Implemented**:

| Algorithm | Use Case | Pros | Cons |
|-----------|----------|------|------|
| **Token Bucket** | Smooth rate limiting with burst | Allows burst traffic | More complex |
| **Leaky Bucket** | Strict rate enforcement | Simple, predictable | No burst support |
| **Sliding Window** | Accurate rate measurement | Precise | Memory intensive |

**Multi-Level Rate Limiting**:
1. **NGINX**: 100 req/s per IP (DDoS protection)
2. **Application**: 100 req/s per user (fair usage)
3. **Database**: Connection pool limits (25 connections)

### 5. Connection Pooling

**Database Connection Pool** ([pool.go](internal/infrastructure/database/pool.go)):

```go
db.SetMaxOpenConns(25)         // Max connections
db.SetMaxIdleConns(5)          // Keep 5 idle
db.SetConnMaxLifetime(5*time.Minute)   // Recycle connections
db.SetConnMaxIdleTime(10*time.Minute)  // Close idle
```

**Why Connection Pooling**:
- **Prevents Exhaustion**: Limits concurrent DB connections
- **Reduces Latency**: Reuses existing connections
- **Connection Lifecycle**: Handles stale connections
- **Monitoring**: Tracks connection metrics

**Connection Pool Metrics**:
```go
stats := db.Stats()
// InUse: Active connections
// Idle: Available connections
// WaitCount: Requests waiting for connection
// WaitDuration: Total wait time
```

### 6. Backpressure Handling

**System Health Monitoring**:

```go
// Backpressure middleware
func (m *BackpressureMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !m.healthChecker() {
            // System overloaded - reject request
            http.Error(w, "Service unavailable", 503)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Health check
func healthChecker() bool {
    // Worker pool < 90% full
    if workerPool.QueueSize() > 0.9 * workerPool.Capacity() {
        return false
    }
    // Database connection pool healthy
    if dbPool.Stats().WaitCount > 100 {
        return false
    }
    return true
}
```

**Backpressure Strategies**:

1. **Worker Pool Queue Full**: Return 503 Service Unavailable
2. **Database Slow**: Increase timeout, enable circuit breaker
3. **Memory High**: Trigger garbage collection, reject requests
4. **Downstream Service Slow**: Circuit breaker opens

**Client Behavior**:
- Receive `503 Service Unavailable`
- Respect `Retry-After: 30` header
- Implement exponential backoff

### 7. Circuit Breaker Pattern

**Implementation** ([breaker.go](internal/infrastructure/circuitbreaker/breaker.go)):

```go
// Three states: CLOSED -> OPEN -> HALF-OPEN -> CLOSED
type State int

const (
    StateClosed    // Normal operation
    StateOpen      // Failing, reject requests
    StateHalfOpen  // Testing recovery
)

// Configuration
circuitBreaker := NewCircuitBreaker(Config{
    MaxFailures:     5,      // Open after 5 failures
    Timeout:         60s,    // Wait 60s before half-open
    HalfOpenSuccess: 2,      // Need 2 successes to close
})

// Execute with protection
err := circuitBreaker.Execute(ctx, func() error {
    return paymentGateway.Process(payment)
})

if err == ErrCircuitOpen {
    // Fallback: return cached response, queue for retry
    return fallbackResponse
}
```

**State Transitions**:

```
CLOSED (Normal)
   │
   ├─ 5 consecutive failures
   │
   ▼
OPEN (Rejecting requests)
   │
   ├─ Wait 60 seconds
   │
   ▼
HALF-OPEN (Testing)
   │
   ├─ 2 successes ─────────> CLOSED
   │
   └─ 1 failure ──────────> OPEN
```

**Failure Scenarios Handled**:
- Payment gateway timeout
- Payment gateway 500 errors
- Database connection failure
- External API unavailable

## 💥 Failure Scenarios and System Response

### Scenario 1: Flash Sale (Traffic Spike)

**Load**: 10,000 requests/second (100x normal)

**System Response**:
```
1. NGINX Rate Limiting
   ├─> Limit to 100 req/s per IP
   └─> Return 429 with Retry-After header

2. Application Rate Limiting
   ├─> Token bucket allows burst (200 req)
   └─> Smooth out traffic to backend

3. Worker Pool Backpressure
   ├─> Queue fills up (1000 tasks)
   └─> Additional requests get 503

4. Horizontal Scaling (Auto-scale)
   ├─> Detect high load
   └─> Spin up additional instances (api-4, api-5)

5. Connection Pool
   ├─> Reuse DB connections
   └─> Queue requests if pool exhausted
```

**Result**: System handles spike gracefully, rejects excess with proper error codes

### Scenario 2: Database Slowdown

**Issue**: Database query latency increases to 5 seconds

**System Response**:
```
1. Connection Pool Monitoring
   ├─> Detect WaitCount increasing
   └─> Log warning

2. Query Timeout
   ├─> Context timeout (30s) prevents hanging
   └─> Return error to client

3. Backpressure Middleware
   ├─> Detect DB connection pool stressed
   └─> Return 503 to new requests

4. Circuit Breaker (if DB completely down)
   ├─> After 5 failures, open circuit
   └─> Return cached data or error immediately

5. Auto-Recovery
   ├─> When DB recovers, circuit enters HALF-OPEN
   └─> Test with limited requests, then CLOSE
```

### Scenario 3: Payment Gateway Failure

**Issue**: External payment gateway returns 500 errors

**System Response**:
```
1. Circuit Breaker Detects Failures
   ├─> Failure 1-4: Log and retry
   └─> Failure 5: Open circuit

2. Circuit OPEN State
   ├─> Stop sending requests to gateway
   └─> Return error immediately (fail fast)

3. Fallback Strategy
   ├─> Return payment status: PENDING
   ├─> Queue for retry (background job)
   └─> Send webhook notification when processed

4. Half-Open Testing (after 60s)
   ├─> Send test request
   ├─> If success: Close circuit
   └─> If failure: Back to OPEN

5. Monitoring & Alerting
   └─> Alert DevOps team about gateway issue
```

### Scenario 4: Memory Pressure

**Issue**: API instance running low on memory

**System Response**:
```
1. Go Garbage Collector
   ├─> Automatic GC runs
   └─> Reclaim unused memory

2. Worker Pool Limits
   ├─> Fixed number of workers (50)
   └─> Bounded queue (1000) prevents OOM

3. Connection Pool Limits
   ├─> Max connections (25)
   └─> Prevents connection leak

4. Container Resource Limits
   ├─> Docker memory limit: 512MB
   └─> OOM killer restarts container if exceeded

5. Kubernetes (if deployed)
   ├─> Detect unhealthy pod
   └─> Replace with new instance
```

### Scenario 5: Cascading Failure Prevention

**Issue**: One service slowdown could affect entire system

**Protection Mechanisms**:

```
┌─────────────────────────────────────────────────────┐
│         Cascading Failure Prevention                 │
├─────────────────────────────────────────────────────┤
│                                                      │
│  1. Timeout on Every External Call                  │
│     └─> context.WithTimeout(30s)                    │
│                                                      │
│  2. Circuit Breaker per External Service            │
│     └─> Payment Gateway, Email Service, etc.        │
│                                                      │
│  3. Worker Pool Isolation                           │
│     └─> Bounded queue prevents unbounded growth     │
│                                                      │
│  4. Connection Pool Limits                          │
│     └─> MaxOpenConns prevents DB exhaustion         │
│                                                      │
│  5. Rate Limiting                                    │
│     └─> Prevents single user from DoS               │
│                                                      │
│  6. Health Checks                                    │
│     └─> Load balancer removes unhealthy instances   │
│                                                      │
│  7. Graceful Degradation                            │
│     └─> Return cached data or partial response      │
│                                                      │
└─────────────────────────────────────────────────────┘
```

## 🚀 Golang-Specific Optimizations

### 1. Goroutine Management

**Worker Pool Pattern** prevents goroutine explosion:

```go
// ❌ BAD: Unlimited goroutines
for _, payment := range payments {
    go processPayment(payment)  // Could spawn millions!
}

// ✅ GOOD: Bounded worker pool
for _, payment := range payments {
    workerPool.Submit(ctx, func() {
        processPayment(payment)
    })
}
```

### 2. Context Propagation

**Timeout and Cancellation**:

```go
// Request-level timeout
ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
defer cancel()

// Propagate to all operations
payment, err := repository.Save(ctx, payment)
result, err := gateway.Process(ctx, payment)

// Context cancellation signals to all goroutines
select {
case <-ctx.Done():
    return ctx.Err()  // Timeout or cancelled
case result := <-resultChan:
    return result
}
```

### 3. Graceful Shutdown

**Drain in-flight requests**:

```go
func main() {
    srv := &http.Server{Addr: ":8080", Handler: router}
    
    // Start server
    go srv.ListenAndServe()
    
    // Wait for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // 1. Stop accepting new requests
    srv.Shutdown(ctx)
    
    // 2. Finish in-flight worker tasks
    workerPool.Shutdown(30 * time.Second)
    
    // 3. Close database connections
    dbPool.Close()
}
```

### 4. Memory Efficiency

**Struct Padding Optimization**:

```go
// ❌ BAD: 24 bytes due to padding
type Payment struct {
    status  int8    // 1 byte + 7 padding
    amount  int64   // 8 bytes
    id      int32   // 4 bytes + 4 padding
}

// ✅ GOOD: 16 bytes (aligned)
type Payment struct {
    amount  int64   // 8 bytes
    id      int32   // 4 bytes
    status  int8    // 1 byte + 3 padding
}
```

**Sync.Pool for Object Reuse**:

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

// Get from pool
buf := bufferPool.Get().(*bytes.Buffer)
defer bufferPool.Put(buf)
buf.Reset()
```

## 📊 Performance Characteristics

### Expected Throughput

| Metric | Value | Notes |
|--------|-------|-------|
| Requests/second | 1,000 - 10,000 | Per instance |
| Latency (p50) | 10ms | Without external calls |
| Latency (p99) | 100ms | With DB + gateway |
| Worker Pool Size | 50 | Tunable |
| Queue Size | 1,000 | Backpressure threshold |
| Max Connections | 25 per instance | Database pool |
| Memory per instance | 512MB | Docker limit |
| CPU per instance | 1.0 core | Docker limit |

### Horizontal Scaling Math

```
Single Instance: 1,000 req/s
Target Load: 10,000 req/s
Required Instances: 10,000 / 1,000 = 10 instances

With 20% overhead: 12 instances
With N+2 redundancy: 14 instances
```

## 🔧 How to Run

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)

### Quick Start

```bash
# Clone repository
cd d:\project\payment-go

# Build and start all services
docker-compose up --build

# Access API
curl http://localhost/api/v1/payments

# View metrics
open http://localhost:9090  # Prometheus
open http://localhost:3000  # Grafana
```

### Development Mode

```bash
# Install dependencies
go mod download

# Run locally
go run cmd/api/main.go

# Run tests
go test ./...

# Load test
hey -n 10000 -c 100 http://localhost:8080/api/v1/payments
```

### Environment Variables

```bash
SERVER_ADDRESS=:8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=payment_db
REDIS_HOST=localhost
REDIS_PORT=6379
```

## 📈 Monitoring & Observability

### Metrics Exposed

```
# Worker Pool
worker_pool_queue_size
worker_pool_tasks_processed_total
worker_pool_tasks_failed_total

# Circuit Breaker
circuit_breaker_state{service="payment_gateway"}
circuit_breaker_failures_total

# Rate Limiter
rate_limiter_requests_total
rate_limiter_requests_denied_total

# Database
db_connections_open
db_connections_idle
db_connections_wait_duration
```

### Health Endpoints

- `GET /health` - Liveness probe
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

## 🎯 Trade-offs & Design Decisions

### 1. Async Processing (Worker Pool)

**Decision**: Process payments asynchronously
- ✅ **Pro**: Non-blocking API, better throughput
- ✅ **Pro**: Natural backpressure mechanism
- ❌ **Con**: Client must poll for status
- ❌ **Con**: Eventual consistency

### 2. Token Bucket Rate Limiting

**Decision**: Allow burst traffic
- ✅ **Pro**: Better user experience during bursts
- ✅ **Pro**: Smooth out traffic
- ❌ **Con**: More complex than fixed window
- ❌ **Con**: Requires more memory

### 3. Circuit Breaker per Service

**Decision**: Isolate failures
- ✅ **Pro**: Prevents cascading failures
- ✅ **Pro**: Fast fail when service is down
- ❌ **Con**: May reject valid requests
- ❌ **Con**: Tuning thresholds is tricky

### 4. PostgreSQL Connection Pool

**Decision**: Limit max connections (25)
- ✅ **Pro**: Prevents database overload
- ✅ **Pro**: Connection reuse
- ❌ **Con**: May block during high load
- ❌ **Con**: Needs careful tuning

## 📚 Key Takeaways

1. **Backpressure is Essential**: Reject load you can't handle gracefully
2. **Context Everywhere**: Propagate timeouts and cancellation
3. **Isolate Failures**: Circuit breakers prevent cascading failures
4. **Bound Everything**: Worker pools, queues, connections
5. **Monitor Everything**: Metrics are critical for production
6. **Graceful Degradation**: Better to return partial data than nothing
7. **Horizontal Scaling**: Stateless design enables easy scaling
8. **Rate Limiting**: Protect against abuse and DoS

---

**Built with ❤️ using Go, Clean Architecture, DDD, and Docker**
