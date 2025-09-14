package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/nohdol/claude-auto/pkg/types"
	"github.com/rs/zerolog"
)

// GitManager manages Git operations
type GitManager struct {
	repo       *git.Repository
	worktree   *git.Worktree
	commitSize types.CommitSize
	author     *object.Signature
	logger     zerolog.Logger
	projectDir string
}

// NewGitManager creates a new Git manager
func NewGitManager(projectDir string, commitSize types.CommitSize, logger zerolog.Logger) (*GitManager, error) {
	// Initialize or open repository
	repo, err := initOrOpenRepo(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	author := &object.Signature{
		Name:  "Claude Auto",
		Email: "claude-auto@example.com",
		When:  time.Now(),
	}

	return &GitManager{
		repo:       repo,
		worktree:   worktree,
		commitSize: commitSize,
		author:     author,
		logger:     logger,
		projectDir: projectDir,
	}, nil
}

// InitRepo initializes a new Git repository
func (gm *GitManager) InitRepo() error {
	if gm.repo != nil {
		return nil // Already initialized
	}

	repo, err := git.PlainInit(gm.projectDir, false)
	if err != nil {
		return fmt.Errorf("failed to init repository: %w", err)
	}

	gm.repo = repo
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}
	gm.worktree = worktree

	gm.logger.Info().Str("path", gm.projectDir).Msg("Git repository initialized")
	return nil
}

// SmartCommit creates commits based on task type and changes
func (gm *GitManager) SmartCommit(files []string, taskType types.TaskType) error {
	if len(files) == 0 {
		return nil // Nothing to commit
	}

	// Analyze changes
	changes := gm.analyzeChanges(files)

	// Generate commit message
	message := gm.generateCommitMessage(taskType, changes)

	// Stage files
	for _, file := range files {
		relPath, err := filepath.Rel(gm.projectDir, file)
		if err != nil {
			gm.logger.Warn().
				Str("file", file).
				Err(err).
				Msg("Failed to get relative path")
			continue
		}

		if _, err := gm.worktree.Add(relPath); err != nil {
			gm.logger.Warn().
				Str("file", relPath).
				Err(err).
				Msg("Failed to stage file")
		}
	}

	// Create commit
	commit, err := gm.worktree.Commit(message, &git.CommitOptions{
		Author: gm.author,
	})
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	gm.logger.Info().
		Str("hash", commit.String()).
		Str("message", message).
		Msg("Commit created")

	return nil
}

// AtomicCommit creates small, atomic commits for each file
func (gm *GitManager) AtomicCommit(file string, taskType types.TaskType) error {
	return gm.SmartCommit([]string{file}, taskType)
}

// BatchCommit creates a single commit for multiple files
func (gm *GitManager) BatchCommit(files []string, taskType types.TaskType, message string) error {
	if len(files) == 0 {
		return nil
	}

	// Stage all files
	for _, file := range files {
		relPath, err := filepath.Rel(gm.projectDir, file)
		if err != nil {
			continue
		}
		gm.worktree.Add(relPath)
	}

	// Use provided message or generate one
	if message == "" {
		changes := gm.analyzeChanges(files)
		message = gm.generateCommitMessage(taskType, changes)
	}

	// Create commit
	_, err := gm.worktree.Commit(message, &git.CommitOptions{
		Author: gm.author,
	})

	return err
}

// analyzeChanges analyzes the changes in files
func (gm *GitManager) analyzeChanges(files []string) *ChangeAnalysis {
	analysis := &ChangeAnalysis{
		Files:      files,
		AddedFiles: 0,
		ModifiedFiles: 0,
		DeletedFiles: 0,
	}

	status, err := gm.worktree.Status()
	if err != nil {
		return analysis
	}

	for _, file := range files {
		relPath, err := filepath.Rel(gm.projectDir, file)
		if err != nil {
			continue
		}

		fileStatus := status.File(relPath)
		switch fileStatus.Staging {
		case git.Added, git.Untracked:
			analysis.AddedFiles++
		case git.Modified:
			analysis.ModifiedFiles++
		case git.Deleted:
			analysis.DeletedFiles++
		}
	}

	// Generate summary
	parts := []string{}
	if analysis.AddedFiles > 0 {
		parts = append(parts, fmt.Sprintf("Added %d files", analysis.AddedFiles))
	}
	if analysis.ModifiedFiles > 0 {
		parts = append(parts, fmt.Sprintf("Modified %d files", analysis.ModifiedFiles))
	}
	if analysis.DeletedFiles > 0 {
		parts = append(parts, fmt.Sprintf("Deleted %d files", analysis.DeletedFiles))
	}

	if len(parts) > 0 {
		analysis.Summary = strings.Join(parts, ", ")
	} else {
		analysis.Summary = "Updated files"
	}

	return analysis
}

// generateCommitMessage generates a commit message based on task type
func (gm *GitManager) generateCommitMessage(taskType types.TaskType, changes *ChangeAnalysis) string {
	prefix := gm.getCommitPrefix(taskType)

	// Use Conventional Commits format
	switch taskType {
	case types.TaskTypeFrontend:
		return fmt.Sprintf("%s: %s", prefix, changes.Summary)
	case types.TaskTypeBackend:
		return fmt.Sprintf("%s: %s", prefix, changes.Summary)
	case types.TaskTypeDatabase:
		return fmt.Sprintf("%s: database schema and migrations", prefix)
	case types.TaskTypeTesting:
		return fmt.Sprintf("%s: add tests", prefix)
	case types.TaskTypeDocumentation:
		return fmt.Sprintf("%s: update documentation", prefix)
	case types.TaskTypeDevOps:
		return fmt.Sprintf("%s: %s", prefix, changes.Summary)
	default:
		return fmt.Sprintf("chore: %s", changes.Summary)
	}
}

// getCommitPrefix returns the conventional commit prefix for a task type
func (gm *GitManager) getCommitPrefix(taskType types.TaskType) string {
	prefixes := map[types.TaskType]string{
		types.TaskTypeFrontend:      "feat(ui)",
		types.TaskTypeBackend:       "feat(api)",
		types.TaskTypeDatabase:      "feat(db)",
		types.TaskTypeTesting:       "test",
		types.TaskTypeDocumentation: "docs",
		types.TaskTypeDevOps:        "ci",
	}

	if prefix, exists := prefixes[taskType]; exists {
		return prefix
	}
	return "chore"
}

// Push pushes commits to remote repository
func (gm *GitManager) Push(remote, branch string) error {
	// Get remote
	r, err := gm.repo.Remote(remote)
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	// Push
	err = r.Push(&git.PushOptions{
		RemoteName: remote,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)),
		},
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push: %w", err)
	}

	gm.logger.Info().
		Str("remote", remote).
		Str("branch", branch).
		Msg("Pushed to remote")

	return nil
}

// CreateBranch creates a new branch
func (gm *GitManager) CreateBranch(name string) error {
	// Get HEAD reference
	head, err := gm.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Create new branch
	ref := fmt.Sprintf("refs/heads/%s", name)

	err = gm.repo.CreateBranch(&config.Branch{
		Name:   name,
		Remote: "origin",
		Merge:  plumbing.ReferenceName(ref),
	})

	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout new branch
	err = gm.worktree.Checkout(&git.CheckoutOptions{
		Branch: head.Name(),
		Create: false,
	})

	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	gm.logger.Info().Str("branch", name).Msg("Branch created")
	return nil
}

// GetCurrentBranch returns the current branch name
func (gm *GitManager) GetCurrentBranch() (string, error) {
	head, err := gm.repo.Head()
	if err != nil {
		return "", err
	}

	return head.Name().Short(), nil
}

// GetStatus returns the current Git status
func (gm *GitManager) GetStatus() (*GitStatus, error) {
	status, err := gm.worktree.Status()
	if err != nil {
		return nil, err
	}

	gitStatus := &GitStatus{
		Modified: []string{},
		Added:    []string{},
		Deleted:  []string{},
		Untracked: []string{},
	}

	for file, s := range status {
		switch s.Staging {
		case git.Modified:
			gitStatus.Modified = append(gitStatus.Modified, file)
		case git.Added:
			gitStatus.Added = append(gitStatus.Added, file)
		case git.Deleted:
			gitStatus.Deleted = append(gitStatus.Deleted, file)
		case git.Untracked:
			gitStatus.Untracked = append(gitStatus.Untracked, file)
		}
	}

	gitStatus.HasChanges = len(gitStatus.Modified) > 0 ||
		len(gitStatus.Added) > 0 ||
		len(gitStatus.Deleted) > 0 ||
		len(gitStatus.Untracked) > 0

	return gitStatus, nil
}

// initOrOpenRepo initializes or opens an existing repository
func initOrOpenRepo(path string) (*git.Repository, error) {
	// Try to open existing repository
	repo, err := git.PlainOpen(path)
	if err == nil {
		return repo, nil
	}

	// If not found, initialize new repository
	if err == git.ErrRepositoryNotExists {
		repo, err = git.PlainInit(path, false)
		if err != nil {
			return nil, err
		}
		return repo, nil
	}

	return nil, err
}

// ChangeAnalysis represents analysis of file changes
type ChangeAnalysis struct {
	Files         []string
	AddedFiles    int
	ModifiedFiles int
	DeletedFiles  int
	Summary       string
}

// GitStatus represents the current Git status
type GitStatus struct {
	Modified  []string
	Added     []string
	Deleted   []string
	Untracked []string
	HasChanges bool
}

// SetAuthor sets the commit author
func (gm *GitManager) SetAuthor(name, email string) {
	gm.author = &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}
}

// AddRemote adds a remote repository
func (gm *GitManager) AddRemote(name, url string) error {
	_, err := gm.repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})

	if err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	gm.logger.Info().
		Str("name", name).
		Str("url", url).
		Msg("Remote added")

	return nil
}

// GetProjectDir returns the project directory
func (gm *GitManager) GetProjectDir() string {
	return gm.projectDir
}