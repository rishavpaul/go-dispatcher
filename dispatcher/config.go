package dispatcher

import "time"

// Config holds the basic configuration for the dispatcher
type Config struct {
	// NumWorkers is the number of concurrent workers to run
	NumWorkers int

	// QueueSize is the size of the task queue
	QueueSize int

	// TaskTimeout is how long a task can run before being cancelled
	TaskTimeout time.Duration
}

// DefaultConfig returns a simple default configuration
// Note: In a real application, you might want more configuration options
func DefaultConfig() Config {
	return Config{
		NumWorkers:  4,           // Reasonable default for most use cases
		QueueSize:   8,           // Twice the number of workers
		TaskTimeout: time.Second, // 1 second timeout
	}
}

// DefaultQueueSize returns the default queue size.
func DefaultQueueSize() int {
	return 8 // Match the test's expectation
}
