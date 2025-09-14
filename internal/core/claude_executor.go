package core

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ClaudeOptions represents options for Claude execution
type ClaudeOptions struct {
	Role            string
	Model           string
	Temperature     float64
	MaxTokens       int
	SystemPrompt    string
	AdditionalFlags []string
}

// ClaudeResponse represents the response from Claude
type ClaudeResponse struct {
	Output      string
	Error       error
	ExitCode    int
	Duration    time.Duration
	RateLimited bool
}

// ClaudeExecutor manages Claude CLI execution
type ClaudeExecutor struct {
	rateLimiter     *RateLimiter
	sessionManager  *SessionManager
	dangerousMode   bool
	maxRetries      int
	timeout         time.Duration
	mu              sync.Mutex
	activeProcesses map[string]*exec.Cmd
	logger          zerolog.Logger
}

// NewClaudeExecutor creates a new Claude executor
func NewClaudeExecutor(logger zerolog.Logger) *ClaudeExecutor {
	return &ClaudeExecutor{
		rateLimiter:     NewRateLimiter(),
		sessionManager:  NewSessionManager(),
		dangerousMode:   true, // Always use --dangerously-skip-permissions
		maxRetries:      3,
		timeout:         5 * time.Minute,
		activeProcesses: make(map[string]*exec.Cmd),
		logger:          logger,
	}
}

// Execute executes a Claude command with the given prompt
func (ce *ClaudeExecutor) Execute(ctx context.Context, prompt string, options *ClaudeOptions) (*ClaudeResponse, error) {
	return ce.executeWithRetry(ctx, prompt, options)
}

// ExecuteWithRole executes a Claude command with a specific role
func (ce *ClaudeExecutor) ExecuteWithRole(ctx context.Context, prompt string, role string) (*ClaudeResponse, error) {
	options := &ClaudeOptions{
		Role: role,
	}
	return ce.executeWithRetry(ctx, prompt, options)
}

// executeWithRetry executes with automatic retry on failure
func (ce *ClaudeExecutor) executeWithRetry(ctx context.Context, prompt string, options *ClaudeOptions) (*ClaudeResponse, error) {
	var lastResponse *ClaudeResponse
	var lastError error

	for attempt := 0; attempt < ce.maxRetries; attempt++ {
		if attempt > 0 {
			ce.logger.Info().
				Int("attempt", attempt+1).
				Msg("Retrying Claude execution")
		}

		response, err := ce.executeOnce(ctx, prompt, options)
		if err == nil && !response.RateLimited {
			return response, nil
		}

		lastResponse = response
		lastError = err

		// Handle rate limiting
		if response != nil && response.RateLimited {
			retryAfter := ParseRetryAfter(response.Output)
			ce.logger.Warn().
				Dur("retry_after", retryAfter).
				Msg("Rate limited, waiting before retry")
			ce.rateLimiter.SetRateLimit(retryAfter)

			// Wait for rate limit to expire
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryAfter):
				continue
			}
		}

		// Exponential backoff for other errors
		if attempt < ce.maxRetries-1 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			ce.logger.Debug().
				Dur("backoff", backoff).
				Msg("Waiting before retry")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}

	return lastResponse, fmt.Errorf("max retries exceeded: %w", lastError)
}

// executeOnce performs a single execution
func (ce *ClaudeExecutor) executeOnce(ctx context.Context, prompt string, options *ClaudeOptions) (*ClaudeResponse, error) {
	// Wait for rate limiting
	if err := ce.rateLimiter.Wait(); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build command
	args := ce.buildArgs(prompt, options)
	cmd := exec.CommandContext(ctx, "claude", args...)

	// Set up pipes
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Track process
	processID := generateProcessID()
	ce.trackProcess(processID, cmd)
	defer ce.untrackProcess(processID)

	// Execute command
	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	output := stdout.String()
	if stderr.String() != "" {
		output += "\n" + stderr.String()
	}

	response := &ClaudeResponse{
		Output:   output,
		Error:    err,
		Duration: duration,
	}

	// Check for rate limiting
	if IsRateLimitError(output) {
		response.RateLimited = true
		retryAfter := ParseRetryAfter(output)
		ce.rateLimiter.SetRateLimit(retryAfter)
	}

	// Get exit code
	if exitErr, ok := err.(*exec.ExitError); ok {
		response.ExitCode = exitErr.ExitCode()
	}

	return response, nil
}

// buildArgs builds command line arguments for Claude
func (ce *ClaudeExecutor) buildArgs(prompt string, options *ClaudeOptions) []string {
	args := []string{}

	// Always use print mode for non-interactive output
	args = append(args, "--print")

	// Always use dangerous mode
	if ce.dangerousMode {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Add options if provided
	if options != nil {
		// Claude CLI doesn't support --role, so we incorporate it into the system prompt
		systemPrompt := options.SystemPrompt
		if options.Role != "" && systemPrompt == "" {
			systemPrompt = fmt.Sprintf("You are a %s.", options.Role)
		} else if options.Role != "" && systemPrompt != "" {
			systemPrompt = fmt.Sprintf("You are a %s. %s", options.Role, systemPrompt)
		}

		if options.Model != "" {
			args = append(args, "--model", options.Model)
		}
		if systemPrompt != "" {
			args = append(args, "--append-system-prompt", systemPrompt)
		}
		args = append(args, options.AdditionalFlags...)
	}

	// Add prompt
	args = append(args, prompt)

	return args
}

// trackProcess tracks an active process
func (ce *ClaudeExecutor) trackProcess(id string, cmd *exec.Cmd) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.activeProcesses[id] = cmd
}

// untrackProcess removes a process from tracking
func (ce *ClaudeExecutor) untrackProcess(id string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	delete(ce.activeProcesses, id)
}

// Cleanup cleans up all active processes
func (ce *ClaudeExecutor) Cleanup() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	var errors []string
	for id, cmd := range ce.activeProcesses {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				errors = append(errors, fmt.Sprintf("failed to kill process %s: %v", id, err))
			}
		}
	}

	ce.activeProcesses = make(map[string]*exec.Cmd)

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}
	return nil
}

// generateProcessID generates a unique process ID
func generateProcessID() string {
	return fmt.Sprintf("claude-%d", time.Now().UnixNano())
}