package dispatcher

// Task represents a unit of work to be processed
type Task struct {
	// ID uniquely identifies this task
	ID int

	// Payload is the data to be processed
	// Note: In a real application, you might want a more specific type
	Payload interface{}
}

// Result represents the outcome of processing a Task
type Result struct {
	// TaskID identifies which task this result is for
	TaskID int

	// Outcome describes what happened with the task
	// Note: In a real application, you might want more specific outcomes
	Outcome string

	// Err is any error that occurred during processing
	Err error
}
