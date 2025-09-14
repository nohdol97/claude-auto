package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nohdol/claude-auto/internal/core"
	"github.com/nohdol/claude-auto/pkg/types"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// ParallelExecutor executes tasks in parallel based on dependencies
type ParallelExecutor struct {
	taskManager    *TaskManager
	claudeExecutor *core.ClaudeExecutor
	maxWorkers     int
	logger         zerolog.Logger
	mu             sync.RWMutex
	activeWorkers  int
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(tm *TaskManager, ce *core.ClaudeExecutor, maxWorkers int, logger zerolog.Logger) *ParallelExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 3 // Default to 3 workers
	}

	return &ParallelExecutor{
		taskManager:    tm,
		claudeExecutor: ce,
		maxWorkers:     maxWorkers,
		logger:         logger,
	}
}

// ExecuteTasks executes all tasks respecting dependencies
func (pe *ParallelExecutor) ExecuteTasks(ctx context.Context) (*types.ExecutionReport, error) {
	startTime := time.Now()

	// Get execution order
	orderedTasks, err := pe.taskManager.GetExecutionOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to get execution order: %w", err)
	}

	// Build dependency graph
	graph := pe.buildDependencyGraph(orderedTasks)

	// Get batches for parallel execution
	batches := pe.topologicalSort(graph)

	report := &types.ExecutionReport{
		TotalTasks: len(orderedTasks),
		StartTime:  startTime,
		Tasks:      orderedTasks,
	}

	// Execute batches
	for i, batch := range batches {
		pe.logger.Info().
			Int("batch", i+1).
			Int("tasks", len(batch)).
			Msg("Executing batch")

		if err := pe.executeBatch(ctx, batch); err != nil {
			pe.logger.Error().
				Err(err).
				Int("batch", i+1).
				Msg("Batch execution failed")
			// Continue with other batches even if one fails
		}

		// Update report
		pe.updateReport(report)
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	return report, nil
}

// executeBatch executes a batch of tasks in parallel
func (pe *ParallelExecutor) executeBatch(ctx context.Context, batch []*types.Task) error {
	// Create error group with context
	g, ctx := errgroup.WithContext(ctx)

	// Create semaphore for worker limit
	sem := make(chan struct{}, pe.maxWorkers)

	for _, task := range batch {
		task := task // Capture loop variable

		g.Go(func() error {
			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Track active workers
			pe.incrementActiveWorkers()
			defer pe.decrementActiveWorkers()

			// Execute task
			return pe.executeTask(ctx, task)
		})
	}

	// Wait for all tasks in batch to complete
	return g.Wait()
}

// executeTask executes a single task
func (pe *ParallelExecutor) executeTask(ctx context.Context, task *types.Task) error {
	pe.logger.Info().
		Str("task_id", task.ID).
		Str("type", string(task.Type)).
		Msg("Starting task execution")

	// Update status to in progress
	if err := pe.taskManager.UpdateTaskStatus(task.ID, types.TaskStatusInProgress); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Build Claude options based on task type
	options := pe.buildClaudeOptions(task)

	// Execute with Claude
	response, err := pe.claudeExecutor.Execute(ctx, task.Prompt, options)
	if err != nil {
		pe.logger.Error().
			Err(err).
			Str("task_id", task.ID).
			Msg("Task execution failed")

		// Update task with error
		pe.taskManager.SetTaskError(task.ID, err)
		return err
	}

	// Set task result
	if err := pe.taskManager.SetTaskResult(task.ID, response.Output); err != nil {
		return fmt.Errorf("failed to set task result: %w", err)
	}

	// Update status to completed
	if err := pe.taskManager.UpdateTaskStatus(task.ID, types.TaskStatusCompleted); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	pe.logger.Info().
		Str("task_id", task.ID).
		Dur("duration", response.Duration).
		Msg("Task completed successfully")

	return nil
}

// buildClaudeOptions builds Claude execution options based on task type
func (pe *ParallelExecutor) buildClaudeOptions(task *types.Task) *core.ClaudeOptions {
	options := &core.ClaudeOptions{}

	// Set role based on task type
	switch task.Type {
	case types.TaskTypeFrontend:
		options.Role = "frontend-developer"
		options.SystemPrompt = "You are an expert frontend developer specializing in modern web frameworks."
	case types.TaskTypeBackend:
		options.Role = "backend-developer"
		options.SystemPrompt = "You are an expert backend developer specializing in API design and server architecture."
	case types.TaskTypeDatabase:
		options.Role = "database-architect"
		options.SystemPrompt = "You are a database architect specializing in schema design and optimization."
	case types.TaskTypeTesting:
		options.Role = "qa-engineer"
		options.SystemPrompt = "You are a QA engineer specializing in test automation and quality assurance."
	case types.TaskTypeDocumentation:
		options.Role = "technical-writer"
		options.SystemPrompt = "You are a technical writer specializing in clear and comprehensive documentation."
	case types.TaskTypeDevOps:
		options.Role = "devops-engineer"
		options.SystemPrompt = "You are a DevOps engineer specializing in CI/CD and infrastructure automation."
	}

	return options
}

// buildDependencyGraph builds a dependency graph from tasks
func (pe *ParallelExecutor) buildDependencyGraph(tasks []*types.Task) map[string][]string {
	graph := make(map[string][]string)

	for _, task := range tasks {
		if _, exists := graph[task.ID]; !exists {
			graph[task.ID] = []string{}
		}
		for _, dep := range task.Dependencies {
			graph[dep] = append(graph[dep], task.ID)
		}
	}

	return graph
}

// topologicalSort performs topological sort to create execution batches
func (pe *ParallelExecutor) topologicalSort(graph map[string][]string) [][]*types.Task {
	// Calculate in-degree for each task
	inDegree := make(map[string]int)
	for node := range graph {
		if _, exists := inDegree[node]; !exists {
			inDegree[node] = 0
		}
		for _, neighbor := range graph[node] {
			inDegree[neighbor]++
		}
	}

	// Initialize with tasks that have no dependencies
	var batches [][]*types.Task
	processed := make(map[string]bool)

	for len(processed) < len(pe.taskManager.GetAllTasks()) {
		var currentBatch []*types.Task

		// Find all tasks that can be executed now
		for _, task := range pe.taskManager.GetAllTasks() {
			if processed[task.ID] {
				continue
			}

			canExecute := true
			for _, dep := range task.Dependencies {
				if !processed[dep] {
					canExecute = false
					break
				}
			}

			if canExecute {
				currentBatch = append(currentBatch, task)
			}
		}

		if len(currentBatch) == 0 {
			break // No more tasks can be executed
		}

		// Mark batch tasks as processed
		for _, task := range currentBatch {
			processed[task.ID] = true
		}

		batches = append(batches, currentBatch)
	}

	return batches
}

// updateReport updates the execution report with current status
func (pe *ParallelExecutor) updateReport(report *types.ExecutionReport) {
	completedTasks := pe.taskManager.GetTasksByStatus(types.TaskStatusCompleted)
	failedTasks := pe.taskManager.GetTasksByStatus(types.TaskStatusFailed)
	skippedTasks := pe.taskManager.GetTasksByStatus(types.TaskStatusSkipped)

	report.CompletedTasks = len(completedTasks)
	report.FailedTasks = len(failedTasks)
	report.SkippedTasks = len(skippedTasks)
}

// incrementActiveWorkers increments the active worker count
func (pe *ParallelExecutor) incrementActiveWorkers() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.activeWorkers++
}

// decrementActiveWorkers decrements the active worker count
func (pe *ParallelExecutor) decrementActiveWorkers() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.activeWorkers--
}

// GetActiveWorkers returns the current number of active workers
func (pe *ParallelExecutor) GetActiveWorkers() int {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.activeWorkers
}