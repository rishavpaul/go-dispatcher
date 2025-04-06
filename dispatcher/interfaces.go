package dispatcher

import (
	"context"
	"time"
)

// TaskProcessor defines how tasks should be processed
type TaskProcessor interface {
	// ProcessTask handles the actual processing of a task
	// It should respect the context's deadline and cancellation
	ProcessTask(ctx context.Context, task Task) Result
}

// ResultHandler defines how results should be handled
type ResultHandler interface {
	// HandleResult processes a task result
	// It should return an error if the result cannot be handled
	HandleResult(result Result) error
}

// DefaultProcessor is the default implementation of TaskProcessor
type DefaultProcessor struct {
	timeout time.Duration
}

// NewDefaultProcessor creates a new DefaultProcessor
func NewDefaultProcessor(timeout time.Duration) *DefaultProcessor {
	return &DefaultProcessor{
		timeout: timeout,
	}
}

// ProcessTask implements TaskProcessor for DefaultProcessor
func (p *DefaultProcessor) ProcessTask(ctx context.Context, task Task) Result {
	// Create a channel for the result
	resultChan := make(chan Result, 1)

	// Process the task in a goroutine
	go func() {
		// Note: In a real application, you would do actual work here
		// This is a simplified example that just sleeps
		time.Sleep(100 * time.Millisecond)

		select {
		case resultChan <- Result{TaskID: task.ID, Outcome: "processed"}:
		case <-ctx.Done():
		}
	}()

	// Wait for either the result or timeout
	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return Result{
			TaskID:  task.ID,
			Outcome: "timeout",
			Err:     NewTaskError(task.ID, ErrTaskCancelled),
		}
	}
}

// ChannelResultHandler is an implementation of ResultHandler that sends results to a channel
type ChannelResultHandler struct {
	results chan Result
}

// NewChannelResultHandler creates a new ChannelResultHandler
func NewChannelResultHandler(results chan Result) *ChannelResultHandler {
	return &ChannelResultHandler{
		results: results,
	}
}

// HandleResult implements ResultHandler for ChannelResultHandler
func (h *ChannelResultHandler) HandleResult(result Result) error {
	select {
	case h.results <- result:
		return nil
	default:
		return ErrTaskQueueFull
	}
}

// Results returns the channel for receiving results
func (h *ChannelResultHandler) Results() <-chan Result {
	return h.results
}
