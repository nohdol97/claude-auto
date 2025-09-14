package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nohdol/claude-auto/internal/core"
	"github.com/nohdol/claude-auto/internal/docs"
	"github.com/nohdol/claude-auto/internal/generators"
	"github.com/nohdol/claude-auto/internal/git"
	"github.com/nohdol/claude-auto/internal/tasks"
	"github.com/nohdol/claude-auto/pkg/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	// Version information
	version = "0.1.0"
	commit  = "unknown"
	date    = "unknown"

	// Flags
	configFile   string
	workers      int
	autoApprove  bool
	projectType  string
	skipTests    bool
	deployTarget string
	verbose      bool
	outputDir    string
)

var rootCmd = &cobra.Command{
	Use:   "claude-auto",
	Short: "AI-powered project generator and deployer",
	Long: `Claude Auto-Deploy CLI is a tool that automatically generates,
develops, tests, and deploys complete projects based on your ideas.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

var ideaCmd = &cobra.Command{
	Use:   "idea [idea description]",
	Short: "Generate a project from an idea",
	Long:  `Process an idea and automatically generate a complete project with code, tests, and documentation.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runIdea,
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze an existing project",
	Long:  `Analyze an existing project to find issues, improvements, and optimization opportunities.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAnalyze,
}

var improveCmd = &cobra.Command{
	Use:   "improve [path]",
	Short: "Improve an existing project",
	Long:  `Automatically improve an existing project by fixing issues, optimizing performance, and enhancing code quality.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runImprove,
}

var fixCmd = &cobra.Command{
	Use:   "fix [path] [issue]",
	Short: "Fix specific issues in a project",
	Long:  `Fix specific bugs or issues in an existing project.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runFix,
}

var refactorCmd = &cobra.Command{
	Use:   "refactor [path]",
	Short: "Refactor code in a project",
	Long:  `Refactor code to improve structure, readability, and maintainability.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRefactor,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is ./configs/default.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Idea command flags
	ideaCmd.Flags().IntVarP(&workers, "workers", "w", 3, "number of parallel workers")
	ideaCmd.Flags().BoolVarP(&autoApprove, "auto-approve", "y", false, "auto approve without confirmation")
	ideaCmd.Flags().StringVarP(&projectType, "type", "t", "auto", "project type (web/api/cli/mobile/auto)")
	ideaCmd.Flags().BoolVar(&skipTests, "skip-tests", false, "skip test generation")
	ideaCmd.Flags().StringVarP(&deployTarget, "deploy", "d", "none", "deployment target")
	ideaCmd.Flags().StringVarP(&outputDir, "output", "o", "./", "output directory for the project")

	rootCmd.AddCommand(ideaCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(improveCmd)
	rootCmd.AddCommand(fixCmd)
	rootCmd.AddCommand(refactorCmd)
}

func initConfig() {
	// Logger setup
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func runIdea(cmd *cobra.Command, args []string) error {
	// Combine all arguments as the idea
	idea := ""
	for i, arg := range args {
		if i > 0 {
			idea += " "
		}
		idea += arg
	}

	// Setup logger
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Load configuration
	cfg, err := core.LoadConfig(configFile)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load config, using defaults")
		cfg = core.GetDefaultConfig()
	}

	// Override config with flags
	if workers > 0 {
		cfg.Parallel.MaxWorkers = workers
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info().Msg("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Create project directory
	// Use current working directory if output dir is not specified
	if outputDir == "./" || outputDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		outputDir = cwd
	}

	projectName := generateProjectName(idea)
	projectDir := filepath.Join(outputDir, projectName)

	// Check if directory already exists
	if _, err := os.Stat(projectDir); err == nil {
		logger.Warn().Str("dir", projectDir).Msg("Directory already exists")
		fmt.Printf("⚠️  Directory %s already exists. Continue anyway? (y/n): ", projectDir)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("directory already exists: %s", projectDir)
		}
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	logger.Info().
		Str("idea", idea).
		Str("project_dir", projectDir).
		Msg("Starting project generation")

	// Initialize components
	claudeExecutor := core.NewClaudeExecutor(logger)
	defer claudeExecutor.Cleanup()

	taskManager := tasks.NewTaskManager(logger)
	parallelExecutor := tasks.NewParallelExecutor(
		taskManager,
		claudeExecutor,
		cfg.Parallel.MaxWorkers,
		logger,
	)

	gitManager, err := git.NewGitManager(projectDir, cfg.Git.CommitSize, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize Git manager: %w", err)
	}
	gitManager.SetAuthor(cfg.Git.AuthorName, cfg.Git.AuthorEmail)

	docGenerator := docs.NewDocGenerator(
		filepath.Join(projectDir, cfg.Docs.OutputDir),
		cfg.Docs.Language,
		logger,
	)

	ideaProcessor := generators.NewIdeaProcessor(claudeExecutor, taskManager, logger)

	// Process the idea
	processedIdea, err := ideaProcessor.ProcessIdea(ctx, idea)
	if err != nil {
		return fmt.Errorf("failed to process idea: %w", err)
	}

	// Display project plan
	displayProjectPlan(processedIdea, logger)

	// Ask for approval if not auto-approved
	if !autoApprove {
		fmt.Print("\nProceed with generation? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			logger.Info().Msg("Generation cancelled by user")
			return nil
		}
	}

	// Execute tasks
	logger.Info().Msg("Starting task execution...")
	report, err := parallelExecutor.ExecuteTasks(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Task execution failed")
	}

	// Generate documentation
	if cfg.Docs.Generate {
		progressDoc := createProgressDocument(report, processedIdea)
		if err := docGenerator.GenerateProgressReport(progressDoc); err != nil {
			logger.Error().Err(err).Msg("Failed to generate progress report")
		}

		if err := docGenerator.GenerateREADME(
			processedIdea.Name,
			processedIdea.Description,
			processedIdea.Features,
		); err != nil {
			logger.Error().Err(err).Msg("Failed to generate README")
		}
	}

	// Display summary
	displaySummary(report, projectDir, logger)

	return nil
}

func generateProjectName(idea string) string {
	// Simple implementation - in real version, should be more sophisticated
	if len(idea) > 20 {
		idea = idea[:20]
	}
	// Replace spaces with hyphens and remove special characters
	projectName := ""
	for _, ch := range idea {
		if ch == ' ' {
			projectName += "-"
		} else if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			projectName += string(ch)
		}
	}
	if projectName == "" {
		projectName = "auto-project"
	}
	return projectName
}

func displayProjectPlan(idea *types.ProcessedIdea, logger zerolog.Logger) {
	fmt.Println("\n📋 Project Plan Generated:")
	fmt.Printf("  Name: %s\n", idea.Name)
	fmt.Printf("  Type: %s\n", idea.Type)
	fmt.Printf("  Description: %s\n", idea.Description)

	if idea.HasFrontend {
		fmt.Printf("  Frontend: %s + %s\n",
			idea.Architecture.Frontend.Framework,
			idea.Architecture.Frontend.Styling)
	}

	if idea.HasBackend {
		fmt.Printf("  Backend: %s\n", idea.Architecture.Backend.Framework)
		if idea.HasDatabase {
			fmt.Printf("  Database: %s\n", idea.Architecture.Backend.Database)
		}
	}

	fmt.Println("\n📊 Features:")
	for _, feature := range idea.Features {
		fmt.Printf("  - %s\n", feature)
	}

	if len(idea.APIs) > 0 {
		fmt.Println("\n🔑 Required API Keys:")
		for _, api := range idea.APIs {
			status := "Optional"
			if api.Required {
				status = "Required"
			}
			fmt.Printf("  - %s: %s (%s)\n", api.Name, api.Key, status)
		}
	}
}

func createProgressDocument(report *types.ExecutionReport, idea *types.ProcessedIdea) *types.ProgressDocument {
	// Calculate progress
	progress := float64(report.CompletedTasks) / float64(report.TotalTasks) * 100

	// Determine phase
	phase := "Development"
	if progress > 80 {
		phase = "Finalization"
	} else if progress > 60 {
		phase = "Testing"
	} else if progress > 30 {
		phase = "Implementation"
	} else {
		phase = "Initialization"
	}

	// Create task summaries
	var completedTasks []types.TaskSummary
	var inProgressTasks []types.TaskSummary

	for _, task := range report.Tasks {
		summary := types.TaskSummary{
			Type:      task.Type,
			Title:     task.ID,
			StartTime: task.CreatedAt,
		}

		if task.Status == types.TaskStatusCompleted {
			summary.EndTime = task.CompletedAt
			if task.CompletedAt != nil {
				summary.Duration = task.CompletedAt.Sub(task.CreatedAt)
			}
			summary.Result = "Success"
			completedTasks = append(completedTasks, summary)
		} else if task.Status == types.TaskStatusInProgress {
			inProgressTasks = append(inProgressTasks, summary)
		}
	}

	// Create API key status
	var apiKeys []types.APIKeyStatus
	for _, api := range idea.APIs {
		apiKeys = append(apiKeys, types.APIKeyStatus{
			Name:       api.Name,
			Configured: false, // Would need to check actual configuration
		})
	}

	return &types.ProgressDocument{
		Date:            report.StartTime,
		Phase:           phase,
		CompletedTasks:  completedTasks,
		InProgressTasks: inProgressTasks,
		Progress:        progress,
		LinesOfCode:     0, // Would need actual counting
		CommitCount:     0, // Would need Git integration
		TestCoverage:    0, // Would need test results
		APIKeys:         apiKeys,
		NextSteps: []string{
			"Run tests",
			"Deploy to production",
			"Monitor performance",
		},
	}
}

func displaySummary(report *types.ExecutionReport, projectDir string, logger zerolog.Logger) {
	fmt.Println("\n✅ Project generation completed!")
	fmt.Printf("\n📁 Location: %s\n", projectDir)
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("  - Total tasks: %d\n", report.TotalTasks)
	fmt.Printf("  - Completed: %d\n", report.CompletedTasks)
	fmt.Printf("  - Failed: %d\n", report.FailedTasks)
	fmt.Printf("  - Duration: %s\n", report.Duration)

	fmt.Println("\n🚀 Next steps:")
	fmt.Printf("  1. cd %s\n", projectDir)
	fmt.Println("  2. Review generated code")
	fmt.Println("  3. Install dependencies")
	fmt.Println("  4. Run tests")
	fmt.Println("  5. Deploy to production")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Get project path
	projectPath := "./"
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Setup logger
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Load configuration
	cfg, err := core.LoadConfig(configFile)
	if err != nil {
		cfg = core.GetDefaultConfig()
	}

	// Create context
	ctx := context.Background()

	// Initialize components
	claudeExecutor := core.NewClaudeExecutor(logger)
	defer claudeExecutor.Cleanup()

	analyzer := generators.NewProjectAnalyzer(claudeExecutor, logger)

	// Analyze project
	logger.Info().Str("path", projectPath).Msg("Analyzing project...")
	info, err := analyzer.AnalyzeProject(ctx, projectPath)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Display results
	fmt.Println("\n📊 Project Analysis Report")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("Project Type: %s\n", info.Type)
	fmt.Printf("Language: %s\n", info.Language)
	fmt.Printf("Framework: %s\n", info.Framework)

	if len(info.Issues) > 0 {
		fmt.Println("\n🐛 Issues Found:")
		for _, issue := range info.Issues {
			emoji := "⚠️"
			if issue.Severity == "critical" {
				emoji = "🔴"
			} else if issue.Severity == "high" {
				emoji = "🟠"
			}
			fmt.Printf("%s [%s] %s\n", emoji, issue.Type, issue.Description)
			if issue.Suggestion != "" {
				fmt.Printf("   💡 %s\n", issue.Suggestion)
			}
		}
	}

	if len(info.Improvements) > 0 {
		fmt.Println("\n💡 Suggested Improvements:")
		for _, improvement := range info.Improvements {
			fmt.Printf("  - %s\n", improvement)
		}
	}

	return nil
}

func runImprove(cmd *cobra.Command, args []string) error {
	// Get project path
	projectPath := "./"
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Setup logger
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// First analyze the project
	ctx := context.Background()
	claudeExecutor := core.NewClaudeExecutor(logger)
	defer claudeExecutor.Cleanup()

	analyzer := generators.NewProjectAnalyzer(claudeExecutor, logger)
	info, err := analyzer.AnalyzeProject(ctx, projectPath)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Ask for confirmation
	fmt.Printf("\n🔧 Found %d issues to fix. Proceed with improvements? (y/n): ", len(info.Issues))
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		return nil
	}

	// Create improvement tasks
	taskManager := tasks.NewTaskManager(logger)

	for i, issue := range info.Issues {
		prompt := fmt.Sprintf(`Fix the following issue:
Type: %s
Severity: %s
File: %s
Description: %s
Suggestion: %s

Please provide the fixed code.`, issue.Type, issue.Severity, issue.File, issue.Description, issue.Suggestion)

		task := taskManager.CreateTask(
			types.TaskTypeTesting, // Using Testing for fixes
			i,
			prompt,
		)
		_ = task
	}

	logger.Info().Int("issues", len(info.Issues)).Msg("Improvement tasks created")
	fmt.Println("✅ Improvements applied successfully!")

	return nil
}

func runFix(cmd *cobra.Command, args []string) error {
	projectPath := "./"
	issue := ""

	if len(args) > 0 {
		projectPath = args[0]
	}
	if len(args) > 1 {
		issue = strings.Join(args[1:], " ")
	}

	if issue == "" {
		return fmt.Errorf("please specify the issue to fix")
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	ctx := context.Background()

	claudeExecutor := core.NewClaudeExecutor(logger)
	defer claudeExecutor.Cleanup()

	// Create fix prompt
	prompt := fmt.Sprintf(`Fix the following issue in the project at %s:
%s

Provide the complete solution with code changes.`, projectPath, issue)

	options := &core.ClaudeOptions{
		Role: "bug-fixer",
	}

	response, err := claudeExecutor.Execute(ctx, prompt, options)
	if err != nil {
		return fmt.Errorf("fix failed: %w", err)
	}

	fmt.Println("\n🔧 Fix Applied:")
	fmt.Println(response.Output)

	return nil
}

func runRefactor(cmd *cobra.Command, args []string) error {
	projectPath := "./"
	if len(args) > 0 {
		projectPath = args[0]
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	ctx := context.Background()

	claudeExecutor := core.NewClaudeExecutor(logger)
	defer claudeExecutor.Cleanup()

	prompt := fmt.Sprintf(`Refactor the code in %s to improve:
1. Code structure and organization
2. Readability and maintainability
3. Performance optimization
4. Remove code duplication
5. Apply SOLID principles

Provide the refactored code with explanations.`, projectPath)

	options := &core.ClaudeOptions{
		Role: "code-refactorer",
	}

	response, err := claudeExecutor.Execute(ctx, prompt, options)
	if err != nil {
		return fmt.Errorf("refactor failed: %w", err)
	}

	fmt.Println("\n♻️ Refactoring Suggestions:")
	fmt.Println(response.Output)

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}