package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nohdol/claude-auto/pkg/types"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Claude   ClaudeConfig   `mapstructure:"claude"`
	Parallel ParallelConfig `mapstructure:"parallel"`
	Git      GitConfig      `mapstructure:"git"`
	Docs     DocsConfig     `mapstructure:"documentation"`
}

// ClaudeConfig represents Claude-related configuration
type ClaudeConfig struct {
	DangerousMode bool   `mapstructure:"dangerous_mode"`
	MaxRetries    int    `mapstructure:"max_retries"`
	Timeout       string `mapstructure:"timeout"`
	Model         string `mapstructure:"model"`
}

// ParallelConfig represents parallel execution configuration
type ParallelConfig struct {
	MaxWorkers   int    `mapstructure:"max_workers"`
	TaskTimeout  string `mapstructure:"task_timeout"`
	BatchSize    int    `mapstructure:"batch_size"`
}

// GitConfig represents Git-related configuration
type GitConfig struct {
	AutoCommit    bool             `mapstructure:"auto_commit"`
	CommitSize    types.CommitSize `mapstructure:"commit_size"`
	PushStrategy  string           `mapstructure:"push_strategy"`
	AuthorName    string           `mapstructure:"author_name"`
	AuthorEmail   string           `mapstructure:"author_email"`
	RemoteName    string           `mapstructure:"remote_name"`
	DefaultBranch string           `mapstructure:"default_branch"`
}

// DocsConfig represents documentation configuration
type DocsConfig struct {
	Language  string `mapstructure:"language"`
	OutputDir string `mapstructure:"output_dir"`
	Generate  bool   `mapstructure:"generate"`
}

// LoadConfig loads configuration from file and environment
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config name and type
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("default")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.claude-auto")
	}

	// Enable environment variable reading
	v.SetEnvPrefix("CLAUDE_AUTO")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we have defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Claude defaults
	v.SetDefault("claude.dangerous_mode", true)
	v.SetDefault("claude.max_retries", 3)
	v.SetDefault("claude.timeout", "5m")
	v.SetDefault("claude.model", "")

	// Parallel execution defaults
	v.SetDefault("parallel.max_workers", 3)
	v.SetDefault("parallel.task_timeout", "10m")
	v.SetDefault("parallel.batch_size", 5)

	// Git defaults
	v.SetDefault("git.auto_commit", true)
	v.SetDefault("git.commit_size", "small")
	v.SetDefault("git.push_strategy", "batch")
	v.SetDefault("git.author_name", "Claude Auto")
	v.SetDefault("git.author_email", "claude-auto@example.com")
	v.SetDefault("git.remote_name", "origin")
	v.SetDefault("git.default_branch", "main")

	// Documentation defaults
	v.SetDefault("documentation.language", "ko")
	v.SetDefault("documentation.output_dir", "./docs/progress")
	v.SetDefault("documentation.generate", true)
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Validate max workers
	if cfg.Parallel.MaxWorkers <= 0 {
		return fmt.Errorf("parallel.max_workers must be greater than 0")
	}

	// Validate commit size
	validCommitSizes := map[types.CommitSize]bool{
		types.CommitSizeAtomic: true,
		types.CommitSizeSmall:  true,
		types.CommitSizeMedium: true,
	}
	if !validCommitSizes[cfg.Git.CommitSize] {
		return fmt.Errorf("invalid git.commit_size: %s", cfg.Git.CommitSize)
	}

	// Validate push strategy
	validPushStrategies := map[string]bool{
		"immediate": true,
		"batch":     true,
		"manual":    true,
	}
	if !validPushStrategies[cfg.Git.PushStrategy] {
		return fmt.Errorf("invalid git.push_strategy: %s", cfg.Git.PushStrategy)
	}

	return nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(cfg *Config, path string) error {
	v := viper.New()

	// Set configuration values
	v.Set("claude", cfg.Claude)
	v.Set("parallel", cfg.Parallel)
	v.Set("git", cfg.Git)
	v.Set("documentation", cfg.Docs)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write configuration file
	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	return &Config{
		Claude: ClaudeConfig{
			DangerousMode: true,
			MaxRetries:    3,
			Timeout:       "5m",
			Model:         "",
		},
		Parallel: ParallelConfig{
			MaxWorkers:  3,
			TaskTimeout: "10m",
			BatchSize:   5,
		},
		Git: GitConfig{
			AutoCommit:    true,
			CommitSize:    types.CommitSizeSmall,
			PushStrategy:  "batch",
			AuthorName:    "Claude Auto",
			AuthorEmail:   "claude-auto@example.com",
			RemoteName:    "origin",
			DefaultBranch: "main",
		},
		Docs: DocsConfig{
			Language:  "ko",
			OutputDir: "./docs/progress",
			Generate:  true,
		},
	}
}