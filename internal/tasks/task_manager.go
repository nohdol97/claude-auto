package tasks

import (
	"fmt"
	"sync"
	"time"

	"github.com/nohdol/claude-auto/pkg/types"
	"github.com/rs/zerolog"
)

// TaskManager manages task creation and dependencies
type TaskManager struct {
	mu       sync.RWMutex
	tasks    map[string]*types.Task
	logger   zerolog.Logger
	idCounter int
}

// NewTaskManager creates a new task manager
func NewTaskManager(logger zerolog.Logger) *TaskManager {
	return &TaskManager{
		tasks:  make(map[string]*types.Task),
		logger: logger,
	}
}

// CreateTask creates a new task
func (tm *TaskManager) CreateTask(taskType types.TaskType, priority int, prompt string) *types.Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.idCounter++
	taskID := fmt.Sprintf("task-%d-%s", tm.idCounter, taskType)

	task := &types.Task{
		ID:           taskID,
		Type:         taskType,
		Priority:     priority,
		Prompt:       prompt,
		Context:      make(map[string]string),
		Dependencies: []string{},
		Status:       types.TaskStatusPending,
		CreatedAt:    time.Now(),
	}

	tm.tasks[taskID] = task
	return task
}

// AddDependency adds a dependency between tasks
func (tm *TaskManager) AddDependency(taskID, dependsOnID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if _, exists := tm.tasks[dependsOnID]; !exists {
		return fmt.Errorf("dependency task %s not found", dependsOnID)
	}

	// Check for circular dependencies
	if tm.wouldCreateCycle(taskID, dependsOnID) {
		return fmt.Errorf("adding dependency would create a cycle")
	}

	task.Dependencies = append(task.Dependencies, dependsOnID)
	return nil
}

// GetTask retrieves a task by ID
func (tm *TaskManager) GetTask(taskID string) (*types.Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[taskID]
	return task, exists
}

// GetAllTasks returns all tasks
func (tm *TaskManager) GetAllTasks() []*types.Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*types.Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetTasksByStatus returns tasks with a specific status
func (tm *TaskManager) GetTasksByStatus(status types.TaskStatus) []*types.Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tasks []*types.Task
	for _, task := range tm.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetTasksByType returns tasks of a specific type
func (tm *TaskManager) GetTasksByType(taskType types.TaskType) []*types.Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tasks []*types.Task
	for _, task := range tm.tasks {
		if task.Type == taskType {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// UpdateTaskStatus updates the status of a task
func (tm *TaskManager) UpdateTaskStatus(taskID string, status types.TaskStatus) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Status = status
	if status == types.TaskStatusCompleted {
		now := time.Now()
		task.CompletedAt = &now
	}

	return nil
}

// SetTaskResult sets the result of a task
func (tm *TaskManager) SetTaskResult(taskID string, result string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Result = result
	return nil
}

// SetTaskError sets an error for a task
func (tm *TaskManager) SetTaskError(taskID string, err error) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Error = err
	task.Status = types.TaskStatusFailed
	return nil
}

// GetReadyTasks returns tasks that are ready to execute
func (tm *TaskManager) GetReadyTasks() []*types.Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var readyTasks []*types.Task
	for _, task := range tm.tasks {
		if task.Status == types.TaskStatusPending && tm.areDependenciesComplete(task) {
			readyTasks = append(readyTasks, task)
		}
	}
	return readyTasks
}

// areDependenciesComplete checks if all dependencies of a task are complete
func (tm *TaskManager) areDependenciesComplete(task *types.Task) bool {
	for _, depID := range task.Dependencies {
		if dep, exists := tm.tasks[depID]; exists {
			if dep.Status != types.TaskStatusCompleted {
				return false
			}
		}
	}
	return true
}

// wouldCreateCycle checks if adding a dependency would create a cycle
func (tm *TaskManager) wouldCreateCycle(taskID, dependsOnID string) bool {
	visited := make(map[string]bool)
	return tm.hasCycleDFS(dependsOnID, taskID, visited)
}

// hasCycleDFS performs depth-first search to detect cycles
func (tm *TaskManager) hasCycleDFS(current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}

	if visited[current] {
		return false
	}
	visited[current] = true

	if task, exists := tm.tasks[current]; exists {
		for _, dep := range task.Dependencies {
			if tm.hasCycleDFS(dep, target, visited) {
				return true
			}
		}
	}

	return false
}

// GetExecutionOrder returns tasks in execution order (topological sort)
func (tm *TaskManager) GetExecutionOrder() ([]*types.Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Build adjacency list
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for id, task := range tm.tasks {
		if _, exists := inDegree[id]; !exists {
			inDegree[id] = 0
		}
		for _, dep := range task.Dependencies {
			graph[dep] = append(graph[dep], id)
			inDegree[id]++
		}
	}

	// Find tasks with no dependencies
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Perform topological sort
	var result []*types.Task
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if task, exists := tm.tasks[current]; exists {
			result = append(result, task)
		}

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != len(tm.tasks) {
		return nil, fmt.Errorf("dependency cycle detected")
	}

	return result, nil
}