# Lab2_MapReduce

Distributed MapReduce Implementation

This repository contains an implementation of a Distributed MapReduce system. The project was developed as part of a lab assignment and demonstrates a fault-tolerant, parallel MapReduce framework, similar to the one described in the MapReduce paper by Google.
Overview

This implementation simulates a distributed MapReduce system, where:

    The Coordinator assigns tasks to Workers.
    Workers execute Map and Reduce tasks and handle failures.
    Intermediate data is stored locally, and results are combined after processing.

The program has been implemented in Go and runs on a single machine with multiple processes. The implementation is designed to handle edge cases like Worker crashes and ensures correctness under parallel execution.
Features

    Task Assignment:
        The Coordinator manages the lifecycle of Map and Reduce tasks, ensuring each task is executed exactly once.

    Fault Tolerance:
        Detects Worker crashes and reassigns unfinished tasks.
        Handles stale tasks if Workers are slow or unresponsive.

    Parallelism:
        Multiple Workers run in parallel to process Map and Reduce tasks, speeding up execution.

    Deterministic Output:
        Ensures that the output from the distributed MapReduce system matches the sequential implementation (mrsequential).

    Tested with Fault Simulation:
        The system passes all automated tests, including crash recovery tests (test-mr.sh).



How It Works
1. Coordinator

    Manages the list of Map and Reduce tasks.
    Assigns tasks to Workers via RPC.
    Monitors task completion and reassigns failed tasks.
    Moves from the Map phase to the Reduce phase after all Map tasks are completed.

2. Worker

    Requests tasks from the Coordinator via RPC.
    Processes Map tasks to generate intermediate files.
    Processes Reduce tasks to generate final output files.
    Handles RPC failures and retries when the Coordinator is unavailable.

3. Intermediate Data

    Map Workers store intermediate data locally.
    The Reduce Workers access this intermediate data to compute final results.

4. Fault Tolerance

    Workers periodically update the Coordinator about their progress.
    The Coordinator reassigns tasks if a Worker is unresponsive for more than 10 seconds.

Getting Started
1. Prerequisites

    Go (1.16 or later)
    A Unix-based system (Linux or macOS)
    Shell and basic command-line tools

2. Build and Run

    Clone the repository:

git clone <repository-url>
cd 6.5840/src/main

Build the wc.so plugin:

go build -race -buildmode=plugin -gcflags="all=-N -l" ../mrapps/wc.go

Run the Coordinator:

go run mrcoordinator.go pg-*.txt

In separate terminals, start one or more Workers:

go run -race -gcflags="all=-N -l" mrworker.go wc.so



Verify the output:

    cat mr-out-* | sort | more

3. Run Automated Tests

To test the system:

bash test-mr.sh

Expected output:

*** PASSED ALL TESTS

Detailed Implementation
Coordinator

    Maintains two lists of tasks: Map and Reduce.
    Tracks task states (Idle, InProgress, Completed).
    Reassigns tasks if they remain InProgress for more than 10 seconds.
    Handles RPC requests from Workers to assign tasks and receive completion reports.

Worker

    Executes Map tasks:
        Reads input files and applies the Map function.
        Writes intermediate key-value pairs to files, partitioned by Reduce task ID.
    Executes Reduce tasks:
        Reads intermediate files produced by Map Workers.
        Applies the Reduce function to generate final output files.
    Implements retry logic to handle temporary Coordinator unavailability.

Example Output

For input files pg-*.txt (Project Gutenberg texts), the output will be key-value pairs indicating word counts:

A 509
ABOUT 2
ACT 8
...

Fault Tolerance
1. Handling Worker Crashes

    If a Worker crashes mid-task, the Coordinator marks the task as Idle and reassigns it to another Worker.

2. Handling Coordinator Unavailability

    Workers retry connecting to the Coordinator if it becomes temporarily unavailable.
    Graceful error handling ensures no data loss.

Future Enhancements

    Distribute the system across multiple machines.
    Add metrics and performance monitoring.
    Implement a secondary Coordinator for fault tolerance.

License

This project is for educational purposes. Feel free to use and modify the code for learning!
