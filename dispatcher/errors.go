package dispatcher

import (
	"errors"
	"fmt"
)

// Error types for the dispatcher
var (
	// ErrDispatcherClosed is returned when trying to submit to a closed dispatcher
	ErrDispatcherClosed = errors.New("dispatcher is closed")

	// ErrTaskCancelled is returned when a task is cancelled due to timeout or shutdown
	ErrTaskCancelled = errors.New("task was cancelled")

	// ErrTaskQueueFull is returned when the task queue is at capacity
	ErrTaskQueueFull = errors.New("task queue is full")
)

// TaskError represents an error that occurred during task processing
type TaskError struct {
	TaskID int
	Err    error
}

// Error implements the error interface
func (e *TaskError) Error() string {
	return fmt.Sprintf("task %d failed: %v", e.TaskID, e.Err)
}

// Unwrap returns the underlying error
func (e *TaskError) Unwrap() error {
	return e.Err
}

// NewTaskError creates a new TaskError
func NewTaskError(taskID int, err error) error {
	return &TaskError{
		TaskID: taskID,
		Err:    err,
	}
}

// IsTaskError checks if an error is a TaskError
func IsTaskError(err error) bool {
	var taskErr *TaskError
	return errors.As(err, &taskErr)
}

// GetTaskID returns the task ID from a TaskError if the error is a TaskError
func GetTaskID(err error) (int, bool) {
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		return taskErr.TaskID, true
	}
	return 0, false
}
