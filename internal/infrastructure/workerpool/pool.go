package workerpool

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	workerPoolActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "workerpool_active_workers",
			Help: "Number of active workers in the pool",
		},
	)

	workerPoolQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "workerpool_queue_size",
			Help: "Number of tasks in the worker pool queue",
		},
	)
)

// Task represents a unit of work to be processed
type Task func(ctx context.Context) error

// Result contains the outcome of a task execution
type Result struct {
	TaskID    string
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

// WorkerPool manages a pool of workers that process tasks concurrently
// This prevents goroutine explosion and provides backpressure
type WorkerPool struct {
	workerCount int
	taskQueue   chan Task
	resultQueue chan Result
	wg          sync.WaitGroup
	logger      *zap.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	metrics     *Metrics
}

type Metrics struct {
	mu                 sync.RWMutex
	tasksProcessed     int64
	tasksSucceeded     int64
	tasksFailed        int64
	averageProcessTime time.Duration
}

// Config for worker pool
type Config struct {
	WorkerCount     int           // Number of concurrent workers
	QueueSize       int           // Size of task queue (for backpressure)
	ShutdownTimeout time.Duration // Graceful shutdown timeout
}

// NewWorkerPool creates a new worker pool
// queueSize provides backpressure - when full, new tasks will block
func NewWorkerPool(cfg Config, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		workerCount: cfg.WorkerCount,
		taskQueue:   make(chan Task, cfg.QueueSize),
		resultQueue: make(chan Result, cfg.QueueSize),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &Metrics{},
	}

	// Start workers
	for i := 0; i < cfg.WorkerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	logger.Info("Worker pool started",
		zap.Int("workers", cfg.WorkerCount),
		zap.Int("queue_size", cfg.QueueSize),
	)

	return wp
}

// worker is the goroutine that processes tasks
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.logger.Info("Worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-wp.ctx.Done():
			wp.logger.Info("Worker shutting down", zap.Int("worker_id", id))
			return

		case task, ok := <-wp.taskQueue:
			if !ok {
				wp.logger.Info("Task queue closed", zap.Int("worker_id", id))
				return
			}

			// Update metrics
			workerPoolActiveWorkers.Inc()
			workerPoolQueueSize.Set(float64(len(wp.taskQueue)))

			// Execute task with timeout from context
			result := wp.executeTask(task)

			// Update metrics after completion
			workerPoolActiveWorkers.Dec()

			// Update metrics
			wp.updateMetrics(result)

			// Send result (non-blocking)
			select {
			case wp.resultQueue <- result:
			default:
				wp.logger.Warn("Result queue full, dropping result")
			}
		}
	}
}

// executeTask runs a task and captures metrics
func (wp *WorkerPool) executeTask(task Task) Result {
	startTime := time.Now()

	// Create a timeout context for the task
	taskCtx, cancel := context.WithTimeout(wp.ctx, 30*time.Second)
	defer cancel()

	// Execute task
	err := task(taskCtx)

	return Result{
		Error:     err,
		StartTime: startTime,
		EndTime:   time.Now(),
	}
}

// Submit submits a task to the worker pool
// Returns error if queue is full (backpressure mechanism)
func (wp *WorkerPool) Submit(ctx context.Context, task Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wp.ctx.Done():
		return context.Canceled
	case wp.taskQueue <- task:
		workerPoolQueueSize.Set(float64(len(wp.taskQueue)))
		return nil
	default:
		// Queue is full - apply backpressure
		wp.logger.Warn("Worker pool queue full, applying backpressure")
		return ErrQueueFull
	}
}

// SubmitBlocking submits a task and blocks until it can be queued
func (wp *WorkerPool) SubmitBlocking(ctx context.Context, task Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wp.ctx.Done():
		return context.Canceled
	case wp.taskQueue <- task:
		return nil
	}
}

// TrySubmit attempts to submit a task without blocking
func (wp *WorkerPool) TrySubmit(task Task) bool {
	select {
	case wp.taskQueue <- task:
		return true
	default:
		return false
	}
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	wp.logger.Info("Shutting down worker pool")

	// Stop accepting new tasks
	close(wp.taskQueue)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.logger.Info("Worker pool shutdown complete")
		close(wp.resultQueue)
		return nil
	case <-time.After(timeout):
		wp.cancel() // Force shutdown
		wp.logger.Warn("Worker pool shutdown timeout, forcing shutdown")
		return ErrShutdownTimeout
	}
}

// GetMetrics returns current metrics
func (wp *WorkerPool) GetMetrics() Metrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return Metrics{
		tasksProcessed:     wp.metrics.tasksProcessed,
		tasksSucceeded:     wp.metrics.tasksSucceeded,
		tasksFailed:        wp.metrics.tasksFailed,
		averageProcessTime: wp.metrics.averageProcessTime,
	}
}

// QueueSize returns current queue size
func (wp *WorkerPool) QueueSize() int {
	return len(wp.taskQueue)
}

// QueueCapacity returns queue capacity
func (wp *WorkerPool) QueueCapacity() int {
	return cap(wp.taskQueue)
}

// IsHealthy checks if worker pool is healthy
func (wp *WorkerPool) IsHealthy() bool {
	queueUtilization := float64(wp.QueueSize()) / float64(wp.QueueCapacity())
	return queueUtilization < 0.9 // Healthy if queue is less than 90% full
}

func (wp *WorkerPool) updateMetrics(result Result) {
	wp.metrics.mu.Lock()
	defer wp.metrics.mu.Unlock()

	wp.metrics.tasksProcessed++
	if result.Error != nil {
		wp.metrics.tasksFailed++
	} else {
		wp.metrics.tasksSucceeded++
	}

	duration := result.EndTime.Sub(result.StartTime)
	// Calculate rolling average
	wp.metrics.averageProcessTime =
		(wp.metrics.averageProcessTime*time.Duration(wp.metrics.tasksProcessed-1) + duration) /
			time.Duration(wp.metrics.tasksProcessed)
}

// Errors
var (
	ErrQueueFull       = &WorkerPoolError{"queue is full"}
	ErrShutdownTimeout = &WorkerPoolError{"shutdown timeout"}
	ErrPoolShutdown    = &WorkerPoolError{"pool is shutdown"}
)

type WorkerPoolError struct {
	message string
}

func (e *WorkerPoolError) Error() string {
	return e.message
}
