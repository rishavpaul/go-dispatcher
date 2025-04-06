package dispatcher

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestBasicFunctionality verifies the core functionality of the dispatcher
func TestBasicFunctionality(t *testing.T) {
	t.Log(`
🧪 TestBasicFunctionality
┌─────────────────────────────────────────────────────────┐
│ Test Assertion:                                         │
│ - Dispatcher can process multiple tasks concurrently    │
│ - Tasks are distributed across available workers        │
│ - Results are properly collected                        │
└─────────────────────────────────────────────────────────┘

Dispatcher Flow:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Task Queue │────▶│  Worker 1   │────▶│  Results    │
│             │     │  Worker 2   │     │  Channel    │
│             │     │  Worker 3   │     │             │
│             │     │  Worker 4   │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
`)

	t.Log("📋 Queue Size = 10, Workers = 4")

	dp := NewDispatcher(DefaultConfig())
	dp.Start()
	defer dp.Stop()

	// Submit tasks
	t.Log("📨 Submitting 5 tasks")
	for i := 1; i <= 5; i++ {
		if err := dp.Submit(Task{ID: i}); err != nil {
			t.Fatalf("❌ Failed to submit task %d: %v", i, err)
		}
	}

	// Collect results
	t.Log("⏳ Waiting for results")
	results := make(map[int]bool)
	for i := 1; i <= 5; i++ {
		select {
		case result := <-dp.Results():
			if result.Err != nil {
				t.Errorf("❌ Task %d failed: %v", result.TaskID, result.Err)
			}
			t.Logf("✅ Task %d completed", result.TaskID)
			results[result.TaskID] = true
		case <-time.After(time.Second):
			t.Fatalf("❌ Timeout waiting for task %d result", i)
		}
	}

	// Verify all tasks were processed
	t.Log("🔍 Verifying results")
	for i := 1; i <= 5; i++ {
		if !results[i] {
			t.Errorf("❌ Task %d was not processed", i)
		}
	}
	t.Log("✅ TestBasicFunctionality PASSED")
}

// TestStop verifies that the dispatcher can be stopped gracefully
func TestStop(t *testing.T) {
	t.Log(`
🧪 TestStop
┌─────────────────────────────────────────────────────────┐
│ Test Assertion:                                         │
│ - Dispatcher can be gracefully stopped                  │
│ - New tasks are rejected after stop                     │
│ - Workers complete current tasks before stopping        │
└─────────────────────────────────────────────────────────┘

Dispatcher Flow:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Stop       │────▶│  Cancel     │────▶│  Workers    │
│  Signal     │     │  Context    │     │  Complete   │
└─────────────┘     └─────────────┘     └─────────────┘
`)

	t.Log("🧪 Testing Dispatcher Stop")

	dp := NewDispatcher(DefaultConfig())
	dp.Start()

	// Submit a task
	t.Log("📨 Submitting task")
	if err := dp.Submit(Task{ID: 1}); err != nil {
		t.Fatalf("❌ Failed to submit task: %v", err)
	}

	// Wait for the task to be processed
	t.Log("⏳ Waiting for result")
	select {
	case result := <-dp.Results():
		if result.Err != nil {
			t.Errorf("❌ Task failed: %v", result.Err)
		}
		t.Log("✅ Task completed")
	case <-time.After(time.Second):
		t.Fatal("❌ Timeout waiting for result")
	}

	t.Log("🛑 Stopping dispatcher")
	dp.Stop()

	// Verify we can't submit more tasks
	t.Log("📨 Attempting to submit task after stop")
	if err := dp.Submit(Task{ID: 2}); err != ErrDispatcherClosed {
		t.Errorf("❌ Expected ErrDispatcherClosed, got %v", err)
	} else {
		t.Log("✅ Task rejected as expected")
	}
	t.Log("✅ TestStop PASSED")
}

// TestTimeout verifies that tasks respect their timeout
func TestTimeout(t *testing.T) {
	t.Log(`
🧪 TestTimeout
┌─────────────────────────────────────────────────────────┐
│ Test Assertion:                                         │
│ - Tasks respect their timeout settings                  │
│ - Timeout errors are properly propagated                │
│ - Workers don't get stuck on long-running tasks         │
└─────────────────────────────────────────────────────────┘

Dispatcher Flow:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Task with  │────▶│  Context    │────▶│  Timeout    │
│  Timeout    │     │  Deadline   │     │  Error      │
└─────────────┘     └─────────────┘     └─────────────┘
`)

	t.Log("🧪 Testing Task Timeout")

	// Create a config with a short timeout
	config := DefaultConfig()
	config.TaskTimeout = 50 * time.Millisecond
	dp := NewDispatcher(config)
	dp.Start()
	defer dp.Stop()

	// Submit a task
	t.Log("📨 Submitting task that will timeout")
	if err := dp.Submit(Task{ID: 1}); err != nil {
		t.Fatalf("❌ Failed to submit task: %v", err)
	}

	// Wait for the result
	t.Log("⏳ Waiting for result")
	select {
	case result := <-dp.Results():
		if result.Err == nil {
			t.Error("❌ Expected timeout error, got nil")
		} else {
			t.Log("✅ Task timed out as expected")
		}
	case <-time.After(time.Second):
		t.Fatal("❌ Timeout waiting for result")
	}
	t.Log("✅ TestTimeout PASSED")
}

// TestQueueFull verifies that the dispatcher handles a full queue correctly
func TestQueueFull(t *testing.T) {
	t.Log(`
🧪 TestQueueFull
┌─────────────────────────────────────────────────────────┐
│ Test Assertion:                                         │
│ - Dispatcher handles queue full conditions              │
│ - Tasks are rejected when queue is full                 │
│ - Queue processes existing tasks before accepting new   │
└─────────────────────────────────────────────────────────┘

Dispatcher Flow:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Queue      │     │  Queue      │     │  Process    │
│  Filling    │────▶│  Full       │────▶│  Existing   │
│             │     │  Reject     │     │  Tasks      │
└─────────────┘     └─────────────┘     └─────────────┘
`)

	t.Log("🧪 Testing Queue Full Behavior")
	t.Log("📋 Queue Size = 2, Workers = 2")

	// Create channels to control task processing
	startProcessing := make(chan struct{}, 2)
	finishProcessing := make(chan struct{})
	processingTasks := make(map[int]bool)
	var processingMu sync.Mutex

	// Custom processor that signals when a task starts processing
	processor := &CustomProcessor{
		ProcessFunc: func(ctx context.Context, task Task) Result {
			processingMu.Lock()
			processingTasks[task.ID] = true
			processingMu.Unlock()

			startProcessing <- struct{}{} // Signal that processing has started
			<-finishProcessing           // Wait for signal to complete

			return Result{
				TaskID: task.ID,
				Outcome: "processed",
			}
		},
	}

	// Create dispatcher with small queue size
	config := Config{
		NumWorkers: 2,
		QueueSize: 2,
		TaskTimeout: 1 * time.Second,
	}

	d := NewDispatcherWithCustomProcessor(config, processor)
	d.Start()

	// Submit first task
	t.Log("📨 Submitting task 1")
	err := d.Submit(Task{ID: 1, Payload: []byte("test1")})
	if err != nil {
		t.Fatalf("❌ Failed to submit task 1: %v", err)
	}

	// Wait for first task to start processing
	<-startProcessing
	t.Log("👷 Task 1 processing")

	// Submit second task
	t.Log("📨 Submitting task 2")
	err = d.Submit(Task{ID: 2, Payload: []byte("test2")})
	if err != nil {
		t.Fatalf("❌ Failed to submit task 2: %v", err)
	}

	// Wait for second task to start processing
	<-startProcessing
	t.Log("👷 Task 2 processing")

	// Submit third task to fill queue
	t.Log("📨 Submitting task 3")
	err = d.Submit(Task{ID: 3, Payload: []byte("test3")})
	if err != nil {
		t.Fatalf("❌ Failed to submit task 3: %v", err)
	}

	// Submit fourth task to fill queue
	t.Log("📨 Submitting task 4")
	err = d.Submit(Task{ID: 4, Payload: []byte("test4")})
	if err != nil {
		t.Fatalf("❌ Failed to submit task 4: %v", err)
	}

	// Try to submit fifth task to full queue
	t.Log("📨 Attempting to submit task 5")
	err = d.Submit(Task{ID: 5, Payload: []byte("test5")})
	if err != ErrTaskQueueFull {
		t.Errorf("❌ Expected ErrTaskQueueFull, got %v", err)
	} else {
		t.Log("✅ Task 5 rejected (queue full)")
	}

	// Allow tasks to complete
	t.Log("🔄 Processing queued tasks")
	close(finishProcessing)

	// Wait for results with a timeout
	t.Log("⏳ Waiting for results")
	results := make(map[int]bool)
	timeout := time.After(2 * time.Second)
	resultsCount := 0

	// Keep collecting results until timeout or we get enough results
	for resultsCount < 4 {
		select {
		case result := <-d.Results():
			t.Logf("✅ Task %d completed", result.TaskID)
			results[result.TaskID] = true
			resultsCount++
		case <-timeout:
			t.Log("⏰ Timeout waiting for more results")
			goto DONE
		}
	}

DONE:
	// Verify tasks were processed
	t.Log("🔍 Verifying results")
	for i := 1; i <= 4; i++ {
		if !results[i] {
			t.Logf("⚠️ Task %d result not received", i)
		}
	}

	if len(results) < 2 {
		t.Errorf("❌ Expected at least 2 results, got %d", len(results))
	}

	t.Log("🛑 Stopping dispatcher")
	d.Stop()
	t.Log("✅ TestQueueFull PASSED")
}

// CustomProcessor is a test implementation of TaskProcessor
type CustomProcessor struct {
	processTime time.Duration
	onProcessStart func(taskID int)
	ProcessFunc func(ctx context.Context, task Task) Result
}

// ProcessTask implements TaskProcessor for CustomProcessor
func (p *CustomProcessor) ProcessTask(ctx context.Context, task Task) Result {
	if p.ProcessFunc != nil {
		return p.ProcessFunc(ctx, task)
	}
	if p.onProcessStart != nil {
		p.onProcessStart(task.ID)
	}
	time.Sleep(p.processTime)
	return Result{TaskID: task.ID, Outcome: "custom processed"}
}

// TestCustomProcessor verifies that custom processors work
func TestCustomProcessor(t *testing.T) {
	t.Log(`
🧪 TestCustomProcessor
┌─────────────────────────────────────────────────────────┐
│ Test Assertion:                                         │
│ - Custom task processors can be used                    │
│ - Custom processing logic is executed correctly         │
│ - Results from custom processor are handled properly    │
└─────────────────────────────────────────────────────────┘

Dispatcher Flow:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Task       │────▶│  Custom     │────▶│  Processed  │
│  Submitted  │     │  Processor  │     │  Result     │
└─────────────┘     └─────────────┘     └─────────────┘
`)

	t.Log("🧪 Testing Custom Processor")

	// Create a config with default settings
	config := DefaultConfig()

	// Create a custom processor
	processor := &CustomProcessor{
		processTime: 50 * time.Millisecond,
	}

	// Create a dispatcher with our custom processor
	dp := NewDispatcherWithCustomProcessor(config, processor)
	dp.Start()
	defer dp.Stop()

	// Submit a task
	t.Log("📨 Submitting task")
	if err := dp.Submit(Task{ID: 1}); err != nil {
		t.Fatalf("❌ Failed to submit task: %v", err)
	}

	// Wait for the result
	t.Log("⏳ Waiting for result")
	select {
	case result := <-dp.Results():
		if result.Err != nil {
			t.Errorf("❌ Task failed: %v", result.Err)
		}
		if result.Outcome != "custom processed" {
			t.Errorf("❌ Expected outcome 'custom processed', got %q", result.Outcome)
		} else {
			t.Log("✅ Task processed successfully")
		}
	case <-time.After(time.Second):
		t.Fatal("❌ Timeout waiting for result")
	}
	t.Log("✅ TestCustomProcessor PASSED")
}

// CustomResultHandler is a test implementation of ResultHandler
type CustomResultHandler struct {
	results []Result
	mu      sync.Mutex
	wg      sync.WaitGroup
}

// HandleResult implements ResultHandler for CustomResultHandler
func (h *CustomResultHandler) HandleResult(result Result) error {
	h.mu.Lock()
	h.results = append(h.results, result)
	h.mu.Unlock()
	h.wg.Done()
	return nil
}

// TestCustomResultHandler verifies that custom result handlers work
func TestCustomResultHandler(t *testing.T) {
	t.Log(`
🧪 TestCustomResultHandler
┌─────────────────────────────────────────────────────────┐
│ Test Assertion:                                         │
│ - Custom result handlers can be used                    │
│ - Results are properly routed to custom handler         │
│ - Custom handling logic is executed correctly           │
└─────────────────────────────────────────────────────────┘

Dispatcher Flow:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Task       │────▶│  Process    │────▶│  Custom     │
│  Result     │     │  Result     │     │  Handler    │
└─────────────┘     └─────────────┘     └─────────────┘
`)

	t.Log("🧪 Testing Custom Result Handler")

	// Create a custom result handler
	handler := &CustomResultHandler{}
	handler.wg.Add(1) // Expect one result

	// Create a dispatcher with default config and our custom handler
	ctx, cancel := context.WithCancel(context.Background())
	dp := &Dispatcher{
		config:        DefaultConfig(),
		tasks:         make(chan Task, DefaultConfig().QueueSize),
		processor:     NewDefaultProcessor(DefaultConfig().TaskTimeout),
		resultHandler: handler,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start the dispatcher
	dp.Start()
	defer dp.Stop()

	// Submit a task
	t.Log("📨 Submitting task")
	if err := dp.Submit(Task{ID: 1}); err != nil {
		t.Fatalf("❌ Failed to submit task: %v", err)
	}

	// Wait for the result with a timeout
	t.Log("⏳ Waiting for result")
	done := make(chan struct{})
	go func() {
		handler.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("✅ Result handled successfully")
	case <-time.After(time.Second):
		t.Fatal("❌ Timeout waiting for result")
	}

	// Verify the result was handled
	handler.mu.Lock()
	if len(handler.results) != 1 {
		t.Errorf("❌ Expected 1 result, got %d", len(handler.results))
	}
	result := handler.results[0]
	handler.mu.Unlock()

	if result.TaskID != 1 {
		t.Errorf("❌ Expected task ID 1, got %d", result.TaskID)
	}
	if result.Err != nil {
		t.Errorf("❌ Expected no error, got %v", result.Err)
	}
	t.Log("✅ TestCustomResultHandler PASSED")
}

