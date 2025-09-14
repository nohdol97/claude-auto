package types

import (
	"time"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeFrontend      TaskType = "frontend"
	TaskTypeBackend       TaskType = "backend"
	TaskTypeDatabase      TaskType = "database"
	TaskTypeTesting       TaskType = "testing"
	TaskTypeDocumentation TaskType = "documentation"
	TaskTypeDevOps        TaskType = "devops"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusSkipped    TaskStatus = "skipped"
)

// Task represents a single task to be executed
type Task struct {
	ID           string            `json:"id"`
	Type         TaskType          `json:"type"`
	Priority     int               `json:"priority"`
	Prompt       string            `json:"prompt"`
	Context      map[string]string `json:"context"`
	Dependencies []string          `json:"dependencies"`
	Status       TaskStatus        `json:"status"`
	Result       string            `json:"result"`
	Error        error             `json:"error,omitempty"`
	RetryCount   int               `json:"retry_count"`
	CreatedAt    time.Time         `json:"created_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// ProcessedIdea represents a processed project idea
type ProcessedIdea struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Type         string            `json:"type"` // web|api|cli|mobile
	Architecture ProjectArchitecture `json:"architecture"`
	Features     []string          `json:"features"`
	APIs         []APIRequirement  `json:"apis"`
	Phases       []ProjectPhase    `json:"phases"`
	HasFrontend  bool              `json:"has_frontend"`
	HasBackend   bool              `json:"has_backend"`
	HasDatabase  bool              `json:"has_database"`
}

// ProjectArchitecture defines the technical architecture
type ProjectArchitecture struct {
	Frontend FrontendArchitecture `json:"frontend,omitempty"`
	Backend  BackendArchitecture  `json:"backend,omitempty"`
}

// FrontendArchitecture defines frontend technical choices
type FrontendArchitecture struct {
	Framework string `json:"framework"` // Next.js|React|Vue
	Styling   string `json:"styling"`   // Tailwind|CSS Modules|Styled Components
	State     string `json:"state"`     // Redux|Zustand|Context API
}

// BackendArchitecture defines backend technical choices
type BackendArchitecture struct {
	Framework string `json:"framework"` // Express|Fastify|Gin
	Database  string `json:"database"`  // PostgreSQL|MongoDB|MySQL
	Cache     string `json:"cache"`     // Redis|Memcached
}

// APIRequirement represents an external API requirement
type APIRequirement struct {
	Name     string `json:"name"`
	Key      string `json:"key"`
	Required bool   `json:"required"`
}

// ProjectPhase represents a phase in project development
type ProjectPhase struct {
	Name  string   `json:"name"`
	Tasks []string `json:"tasks"`
}

// ExecutionReport represents the result of task execution
type ExecutionReport struct {
	TotalTasks      int           `json:"total_tasks"`
	CompletedTasks  int           `json:"completed_tasks"`
	FailedTasks     int           `json:"failed_tasks"`
	SkippedTasks    int           `json:"skipped_tasks"`
	Duration        time.Duration `json:"duration"`
	Tasks           []*Task       `json:"tasks"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
}

// CommitSize represents the size of commits
type CommitSize string

const (
	CommitSizeAtomic CommitSize = "atomic"
	CommitSizeSmall  CommitSize = "small"
	CommitSizeMedium CommitSize = "medium"
)

// ValidationResult represents the result of build validation
type ValidationResult struct {
	Passed bool     `json:"passed"`
	Checks []Check  `json:"checks"`
	Errors []string `json:"errors"`
}

// Check represents a single validation check
type Check struct {
	Name        string `json:"name"`
	Passed      bool   `json:"passed"`
	Message     string `json:"message"`
	CanAutoFix  bool   `json:"can_auto_fix"`
	AutoFixed   bool   `json:"auto_fixed"`
}

// ProgressDocument represents a progress report
type ProgressDocument struct {
	Date            time.Time        `json:"date"`
	Phase           string           `json:"phase"`
	CompletedTasks  []TaskSummary    `json:"completed_tasks"`
	InProgressTasks []TaskSummary    `json:"in_progress_tasks"`
	Progress        float64          `json:"progress"`
	LinesOfCode     int              `json:"lines_of_code"`
	CommitCount     int              `json:"commit_count"`
	TestCoverage    float64          `json:"test_coverage"`
	APIKeys         []APIKeyStatus   `json:"api_keys"`
	NextSteps       []string         `json:"next_steps"`
}

// TaskSummary represents a summary of a task for reporting
type TaskSummary struct {
	Type          TaskType      `json:"type"`
	Title         string        `json:"title"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       *time.Time    `json:"end_time,omitempty"`
	EstimatedTime *time.Time    `json:"estimated_time,omitempty"`
	Duration      time.Duration `json:"duration,omitempty"`
	Result        string        `json:"result"`
}

// APIKeyStatus represents the status of an API key
type APIKeyStatus struct {
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
}