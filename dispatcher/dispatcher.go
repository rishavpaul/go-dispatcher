package dispatcher

import (
	"context" // provides context management for controlling goroutines
	"log"     // logging package to output log messages
	"sync"    // synchronization primitives such as Mutex and WaitGroup
)

/*
Dispatcher struct holds the configuration and state for task dispatching.
It encapsulates channels, processors, context control, synchronization primitives, and state flags.
*/
type Dispatcher struct {
	config        Config             // Configuration parameters for the dispatcher (e.g., queue size, worker count, etc.)
	tasks         chan Task          // Buffered channel used to queue tasks for processing
	processor     TaskProcessor      // Interface responsible for processing individual tasks
	resultHandler ResultHandler      // Interface to handle and deliver task processing results
	ctx           context.Context    // Context to control cancellation and deadlines across goroutines
	cancel        context.CancelFunc // Function to cancel the above context, stopping all processing
	wg            sync.WaitGroup     // WaitGroup to wait for all worker goroutines to finish before shutdown
	mu            sync.Mutex         // Mutex to ensure safe concurrent access to shared fields (like 'closed')
	closed        bool               // Flag to indicate if the dispatcher has been stopped
}

/*
NewDispatcher creates a new Dispatcher instance with a default task processor.
It uses the given configuration and initializes the dispatcher with a default processor
that uses the task timeout specified in the configuration.
*/
func NewDispatcher(config Config) *Dispatcher {
	return NewDispatcherWithCustomProcessor(config, NewDefaultProcessor(config.TaskTimeout))
}

/*
NewDispatcherWithCustomProcessor creates a new Dispatcher instance using a custom task processor.
It initializes the context, task channel, and result handler based on the provided configuration and processor.
Parameters:
  - config: the configuration settings for the dispatcher.
  - processor: the custom task processor to process tasks.

Returns a pointer to the newly created Dispatcher.
*/
func NewDispatcherWithCustomProcessor(config Config, processor TaskProcessor) *Dispatcher {
	// Create a new cancellable context derived from the background context.
	ctx, cancel := context.WithCancel(context.Background())
	// Create a buffered channel for results based on the configured queue size.
	results := make(chan Result, config.QueueSize)

	// Initialize and return the Dispatcher instance.
	return &Dispatcher{
		config:        config,                            // Save the provided configuration.
		tasks:         make(chan Task, config.QueueSize), // Create a buffered channel for tasks.
		processor:     processor,                         // Set the custom task processor.
		resultHandler: NewChannelResultHandler(results),  // Initialize the result handler with the results channel.
		ctx:           ctx,                               // Save the created context.
		cancel:        cancel,                            // Save the cancel function to stop the context.
		// wg, mu, and closed are zero-initialized by default.
	}
}

/*
Start initiates the dispatcher by starting the worker goroutines.
It locks access to the dispatcher state to prevent race conditions and then starts
a specified number of workers based on the configuration. Each worker listens for tasks
and processes them concurrently.
*/
func (d *Dispatcher) Start() {
	// Lock to ensure that starting process is thread-safe.
	d.mu.Lock()
	// Ensure the lock is released when the function returns.
	defer d.mu.Unlock()

	// If the dispatcher is already closed, do not start workers.
	if d.closed {
		return
	}

	// Log the starting message with the number of workers.
	log.Printf("[DISPATCHER] Starting with %d workers", d.config.NumWorkers)

	// Loop to start each worker as a separate goroutine.
	for i := 0; i < d.config.NumWorkers; i++ {
		// Increment the WaitGroup counter for each worker.
		// WaitGroup tracks active workers: Add(1) for each worker, Done() when worker finishes
		// This ensures graceful shutdown by waiting for all workers to complete
		d.wg.Add(1)
		// Launch a worker goroutine, passing a worker ID (starting from 1).
		go d.worker(i + 1)
	}
}

/*
Stop gracefully shuts down the dispatcher.
It locks the dispatcher state, sets the closed flag, cancels the context, closes the tasks channel,
waits for all worker goroutines to finish, and finally closes the results channel if applicable.
*/
func (d *Dispatcher) Stop() {
	// Lock to ensure that stop process is thread-safe.
	d.mu.Lock()
	// If the dispatcher is already closed, release the lock and return immediately.
	if d.closed {
		d.mu.Unlock()
		return
	}
	// Mark the dispatcher as closed.
	d.closed = true
	// Unlock after updating the state.
	d.mu.Unlock()

	// Log the stopping message.
	log.Printf("[DISPATCHER] Stopping")

	// Cancel the context to signal all workers to stop.
	d.cancel()
	// Close the tasks channel so that workers stop receiving new tasks.
	close(d.tasks)
	// Wait for all workers to finish processing.
	d.wg.Wait()

	// If the result handler is of type ChannelResultHandler, close its results channel.
	if handler, ok := d.resultHandler.(*ChannelResultHandler); ok {
		close(handler.results)
	}
}

/*
Submit enqueues a task into the dispatcher for processing.
It first checks if the dispatcher is closed, then attempts to send the task to the tasks channel.
If the task is successfully enqueued, it logs the submission; otherwise, it returns an error
indicating whether the dispatcher is closed or the task queue is full.

Parameters:
  - task: the Task object to be processed.

Returns an error if submission fails (e.g., dispatcher is closed or the task queue is full).
*/
func (d *Dispatcher) Submit(task Task) error {
	// Lock the mutex to check the closed flag in a thread-safe manner.
	d.mu.Lock()
	// If the dispatcher is closed, unlock and return an error.
	if d.closed {
		d.mu.Unlock()
		return ErrDispatcherClosed
	}
	// Unlock after checking.
	d.mu.Unlock()

	// Use a select statement to attempt task submission without blocking indefinitely.
	select {
	// Attempt to enqueue the task in the tasks channel.
	case d.tasks <- task:
		// Log that the task has been submitted.
		log.Printf("[DISPATCHER] Task %d submitted", task.ID)
		return nil
	// If the context is done (canceled or deadline exceeded), return a closed error.
	case <-d.ctx.Done():
		return ErrDispatcherClosed
	// If the task channel is full, fall into the default case and return an error.
	default:
		return ErrTaskQueueFull
	}
}

// Results returns a read-only channel from which processed task results can be received.
// It casts the result handler to a ChannelResultHandler type if possible, which provides the Results channel.
// If not available, it returns nil.
func (d *Dispatcher) Results() <-chan Result {
	// Check if the result handler is of type ChannelResultHandler.
	if handler, ok := d.resultHandler.(*ChannelResultHandler); ok {
		// Return the channel from which results can be read.
		return handler.Results()
	}
	// If not, return nil.
	return nil
}

/*
worker is the main loop for each worker goroutine.
Each worker listens for tasks on the tasks channel or for cancellation signals from the context.
When a task is received, the worker processes the task with a timeout and sends the result
to the result handler. If the context is done or the tasks channel is closed, the worker exits.

Parameters:
  - id: an integer identifier for the worker (used in logging for clarity).
*/
func (d *Dispatcher) worker(id int) {
	// Ensure that when the worker function exits, it decrements the WaitGroup counter.
	defer d.wg.Done()
	// Log that the worker has started.
	log.Printf("[WORKER-%d] Started", id)

	// Infinite loop: continuously process tasks or exit if the context is canceled.
	for {
		select {
		// Check if the context has been canceled, signaling the worker to stop.
		case <-d.ctx.Done():
			log.Printf("[WORKER-%d] Stopping", id)
			return

		// Listen on the tasks channel for new tasks.
		case task, ok := <-d.tasks:
			// If the tasks channel is closed (ok is false), log and exit the worker.
			if !ok {
				log.Printf("[WORKER-%d] Task channel closed", id)
				return
			}

			// Log that the worker is processing a specific task.
			log.Printf("[WORKER-%d] Processing task %d", id, task.ID)

			// Create a new context with a timeout for processing this task.
			ctx, cancel := context.WithTimeout(d.ctx, d.config.TaskTimeout)
			// Process the task using the processor.
			result := d.processor.ProcessTask(ctx, task)
			// Cancel the taks specific context to free resources.
			cancel()

			// Handle the result using the result handler. If an error occurs, log it.
			if err := d.resultHandler.HandleResult(result); err != nil {
				log.Printf("[WORKER-%d] Failed to handle result for task %d: %v", id, task.ID, err)
			} else {
				// Log successful result handling.
				log.Printf("[WORKER-%d] Sent result for task %d", id, task.ID)
			}
		}
	}
}
