# Go Dispatcher

A concurrent task processing system that efficiently manages and executes tasks using a worker pool pattern.

## System Overview

The Go Dispatcher is a robust concurrent task processing system that implements a worker pool pattern with careful attention to thread safety and graceful shutdown. At its core, it manages a pool of worker goroutines that process tasks concurrently while maintaining proper synchronization and error handling.

The system revolves around a central `Dispatcher` struct that orchestrates task distribution, worker management, and result handling. Tasks are submitted through a buffered channel, which acts as a queue, allowing for controlled backpressure when the system is under heavy load. Each worker goroutine independently picks up tasks from this channel, processes them with a configurable timeout, and sends results through a flexible result handling mechanism.

Concurrency control is implemented through a combination of mutexes for state protection and WaitGroups for worker tracking. The system supports graceful shutdown through context cancellation, ensuring that in-flight tasks are completed before the system terminates. Error handling is comprehensive, covering queue full conditions, dispatcher closed states, task timeouts, and processing errors.

The architecture is designed to be both efficient and safe, with proper synchronization primitives at every step of the task lifecycle. It's also highly configurable, allowing users to specify the number of workers, queue size, and task timeout duration. Additionally, the system supports customization through interfaces for both task processing and result handling, making it adaptable to various use cases.

## Detailed Operation Flow
The following diagram shows the detailed interaction between components during the dispatcher's lifecycle:

```mermaid
sequenceDiagram
    participant Client
    participant Dispatcher
    participant TasksChannel
    participant Worker1
    participant Worker2
    participant ResultHandler

    %% Start sequence
    rect rgb(200, 255, 200)
        Note over Dispatcher: Start() called
        Dispatcher->>Dispatcher: Lock mutex
        Dispatcher->>Dispatcher: Check if closed
        Dispatcher->>Dispatcher: Start workers
        loop For each worker
            Dispatcher->>Worker1: go worker(i+1)
            Dispatcher->>Worker2: go worker(i+1)
            Note over Dispatcher,Worker2: wg.Add(1) for each worker
        end
        Dispatcher->>Dispatcher: Unlock mutex
    end

    %% Submit sequence
    rect rgb(200, 200, 255)
        Note over Client: Submit() called
        Client->>Dispatcher: Submit(task)
        Dispatcher->>Dispatcher: Lock mutex
        Dispatcher->>Dispatcher: Check if closed
        Dispatcher->>Dispatcher: Unlock mutex
        alt Task accepted
            Dispatcher->>TasksChannel: task <- channel
            Note over Dispatcher: Log task submission
        else Queue full
            Dispatcher-->>Client: ErrTaskQueueFull
        else Dispatcher closed
            Dispatcher-->>Client: ErrDispatcherClosed
        end
    end

    %% Worker processing sequence
    rect rgb(255, 200, 200)
        loop Worker processing loop
            Worker1->>TasksChannel: Listen for tasks
            alt Task received
                TasksChannel->>Worker1: task
                Worker1->>Worker1: Create timeout context
                Worker1->>Worker1: Process task
                Worker1->>ResultHandler: Handle result
                Note over Worker1: Log processing
            else Channel closed
                Worker1->>Worker1: Exit
            else Context cancelled
                Worker1->>Worker1: Exit
            end
        end
    end

    %% Stop sequence
    rect rgb(255, 255, 200)
        Note over Dispatcher: Stop() called
        Dispatcher->>Dispatcher: Lock mutex
        Dispatcher->>Dispatcher: Set closed = true
        Dispatcher->>Dispatcher: Unlock mutex
        Dispatcher->>Dispatcher: Cancel context
        Dispatcher->>TasksChannel: Close channel
        Dispatcher->>Dispatcher: wg.Wait()
        Note over Dispatcher: Wait for all workers
        Dispatcher->>ResultHandler: Close results channel
    end
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request
