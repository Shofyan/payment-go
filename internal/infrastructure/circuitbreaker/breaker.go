package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	circuitBreakerStateChanges = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuitbreaker_state_changes_total",
			Help: "Total number of circuit breaker state changes",
		},
		[]string{"from", "to"},
	)
)

// State represents the circuit breaker state
type State int

const (
	StateClosed   State = iota // Normal operation
	StateOpen                  // Failing, reject requests
	StateHalfOpen              // Testing if service recovered
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker prevents cascading failures by stopping requests to failing services
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	lastStateChange time.Time

	// Configuration
	maxFailures     int           // Failures before opening circuit
	timeout         time.Duration // How long to wait in open state
	halfOpenSuccess int           // Successes needed to close from half-open

	// Callbacks
	onStateChange func(from, to State)
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures     int           // Number of failures before opening (default: 5)
	Timeout         time.Duration // Time to wait before half-open (default: 60s)
	HalfOpenSuccess int           // Successes to close from half-open (default: 2)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		MaxFailures:     5,
		Timeout:         60 * time.Second,
		HalfOpenSuccess: 2,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	if cfg.MaxFailures == 0 {
		cfg = DefaultConfig()
	}

	return &CircuitBreaker{
		state:           StateClosed,
		maxFailures:     cfg.MaxFailures,
		timeout:         cfg.Timeout,
		halfOpenSuccess: cfg.HalfOpenSuccess,
		lastStateChange: time.Now(),
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if circuit breaker allows the request
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Allow request
		return nil

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastStateChange) > cb.timeout {
			// Transition to half-open
			cb.setState(StateHalfOpen)
			return nil
		}
		// Circuit is open, reject request
		return ErrCircuitOpen

	case StateHalfOpen:
		// Allow limited requests to test if service recovered
		return nil

	default:
		return ErrCircuitOpen
	}
}

// afterRequest records the result
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.failureCount >= cb.maxFailures {
			cb.setState(StateOpen)
		}

	case StateHalfOpen:
		// Failed during testing, back to open
		cb.setState(StateOpen)
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		// Reset failure count on success
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		// Check if we have enough successes to close
		if cb.successCount >= cb.halfOpenSuccess {
			cb.setState(StateClosed)
		}
	}
}

// setState transitions to a new state
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Record metric
	circuitBreakerStateChanges.WithLabelValues(oldState.String(), newState.String()).Inc()

	// Reset counters based on new state
	switch newState {
	case StateClosed:
		cb.failureCount = 0
		cb.successCount = 0
	case StateOpen:
		cb.successCount = 0
	case StateHalfOpen:
		cb.successCount = 0
		cb.failureCount = 0
	}

	// Trigger callback
	if cb.onStateChange != nil {
		cb.onStateChange(oldState, newState)
	}
}

// GetState returns current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns current metrics
func (cb *CircuitBreaker) GetMetrics() Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Metrics{
		State:           cb.state,
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailureTime: cb.lastFailureTime,
		LastStateChange: cb.lastStateChange,
	}
}

// OnStateChange sets callback for state changes
func (cb *CircuitBreaker) OnStateChange(fn func(from, to State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(StateClosed)
}

// Metrics holds circuit breaker metrics
type Metrics struct {
	State           State
	FailureCount    int
	SuccessCount    int
	LastFailureTime time.Time
	LastStateChange time.Time
}

// Errors
var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

// MultiCircuitBreaker manages multiple circuit breakers by key
type MultiCircuitBreaker struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   Config
}

// NewMultiCircuitBreaker creates a circuit breaker manager
func NewMultiCircuitBreaker(config Config) *MultiCircuitBreaker {
	return &MultiCircuitBreaker{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// Execute executes function with circuit breaker for specific key
func (mcb *MultiCircuitBreaker) Execute(ctx context.Context, key string, fn func() error) error {
	breaker := mcb.getBreaker(key)
	return breaker.Execute(ctx, fn)
}

// getBreaker returns or creates a circuit breaker for key
func (mcb *MultiCircuitBreaker) getBreaker(key string) *CircuitBreaker {
	mcb.mu.RLock()
	breaker, exists := mcb.breakers[key]
	mcb.mu.RUnlock()

	if exists {
		return breaker
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	// Double-check
	breaker, exists = mcb.breakers[key]
	if exists {
		return breaker
	}

	breaker = NewCircuitBreaker(mcb.config)
	mcb.breakers[key] = breaker

	return breaker
}

// GetState returns state for a key
func (mcb *MultiCircuitBreaker) GetState(key string) State {
	breaker := mcb.getBreaker(key)
	return breaker.GetState()
}
