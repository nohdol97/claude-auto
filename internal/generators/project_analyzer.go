package generators

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nohdol/claude-auto/internal/core"
	"github.com/rs/zerolog"
)

// ProjectAnalyzer analyzes existing projects
type ProjectAnalyzer struct {
	claudeExecutor *core.ClaudeExecutor
	logger         zerolog.Logger
}

// NewProjectAnalyzer creates a new project analyzer
func NewProjectAnalyzer(ce *core.ClaudeExecutor, logger zerolog.Logger) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		claudeExecutor: ce,
		logger:         logger,
	}
}

// ProjectInfo contains information about the analyzed project
type ProjectInfo struct {
	Path          string
	Type          string   // web, api, cli, mobile
	Language      string   // javascript, typescript, go, python, etc.
	Framework     string   // react, vue, express, gin, etc.
	HasTests      bool
	TestCoverage  float64
	Dependencies  []string
	Issues        []Issue
	Improvements  []string
	Structure     map[string]int // file counts by type
}

// Issue represents a problem found in the project
type Issue struct {
	Type        string // bug, security, performance, quality
	Severity    string // critical, high, medium, low
	File        string
	Line        int
	Description string
	Suggestion  string
}

// AnalyzeProject analyzes an existing project
func (pa *ProjectAnalyzer) AnalyzeProject(ctx context.Context, projectPath string) (*ProjectInfo, error) {
	pa.logger.Info().Str("path", projectPath).Msg("Analyzing project")

	// Check if project exists
	if _, err := os.Stat(projectPath); err != nil {
		return nil, fmt.Errorf("project path does not exist: %w", err)
	}

	// Scan project structure
	structure := pa.scanProjectStructure(projectPath)

	// Detect project type and framework
	projectType, language, framework := pa.detectProjectType(projectPath, structure)

	// Read key files for analysis
	keyFiles := pa.getKeyFiles(projectPath, projectType)
	fileContents := pa.readKeyFiles(keyFiles)

	// Build analysis prompt
	prompt := pa.buildAnalysisPrompt(projectPath, structure, fileContents)

	// Get analysis from Claude
	options := &core.ClaudeOptions{
		Role:         "code-reviewer",
		SystemPrompt: "You are an expert code reviewer and software architect.",
	}

	response, err := pa.claudeExecutor.Execute(ctx, prompt, options)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	// Parse analysis results
	info := &ProjectInfo{
		Path:      projectPath,
		Type:      projectType,
		Language:  language,
		Framework: framework,
		Structure: structure,
	}

	// Parse issues and improvements from response
	pa.parseAnalysisResults(response.Output, info)

	return info, nil
}

// scanProjectStructure scans the project directory structure
func (pa *ProjectAnalyzer) scanProjectStructure(projectPath string) map[string]int {
	structure := make(map[string]int)

	filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories and node_modules
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || d.Name() == "vendor") {
			return filepath.SkipDir
		}

		if !d.IsDir() {
			ext := filepath.Ext(path)
			structure[ext]++
		}

		return nil
	})

	return structure
}

// detectProjectType detects the project type based on files
func (pa *ProjectAnalyzer) detectProjectType(projectPath string, structure map[string]int) (projectType, language, framework string) {
	// Check for package.json
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		language = "javascript"

		// Read package.json to detect framework
		content, _ := os.ReadFile(filepath.Join(projectPath, "package.json"))
		contentStr := string(content)

		if strings.Contains(contentStr, "react") {
			framework = "react"
			projectType = "web"
		} else if strings.Contains(contentStr, "vue") {
			framework = "vue"
			projectType = "web"
		} else if strings.Contains(contentStr, "express") {
			framework = "express"
			projectType = "api"
		} else if strings.Contains(contentStr, "next") {
			framework = "nextjs"
			projectType = "web"
		}

		if structure[".ts"] > 0 || structure[".tsx"] > 0 {
			language = "typescript"
		}
	}

	// Check for go.mod
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		language = "go"
		projectType = "api"

		// Check for web frameworks
		content, _ := os.ReadFile(filepath.Join(projectPath, "go.mod"))
		if strings.Contains(string(content), "gin") {
			framework = "gin"
		} else if strings.Contains(string(content), "fiber") {
			framework = "fiber"
		} else if strings.Contains(string(content), "echo") {
			framework = "echo"
		}
	}

	// Check for requirements.txt or setup.py
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		language = "python"

		content, _ := os.ReadFile(filepath.Join(projectPath, "requirements.txt"))
		if strings.Contains(string(content), "django") {
			framework = "django"
			projectType = "web"
		} else if strings.Contains(string(content), "flask") {
			framework = "flask"
			projectType = "api"
		} else if strings.Contains(string(content), "fastapi") {
			framework = "fastapi"
			projectType = "api"
		}
	}

	if projectType == "" {
		projectType = "unknown"
	}

	return projectType, language, framework
}

// getKeyFiles returns the key files to analyze based on project type
func (pa *ProjectAnalyzer) getKeyFiles(projectPath string, projectType string) []string {
	var keyFiles []string

	// Common files
	commonFiles := []string{
		"README.md",
		"package.json",
		"go.mod",
		"requirements.txt",
		".env.example",
	}

	for _, file := range commonFiles {
		fullPath := filepath.Join(projectPath, file)
		if _, err := os.Stat(fullPath); err == nil {
			keyFiles = append(keyFiles, fullPath)
		}
	}

	// Add main source files based on project type
	switch projectType {
	case "web":
		candidates := []string{
			"src/App.js", "src/App.jsx", "src/App.tsx",
			"src/index.js", "src/index.jsx", "src/index.tsx",
			"pages/index.js", "pages/index.jsx", "pages/index.tsx",
		}
		for _, file := range candidates {
			fullPath := filepath.Join(projectPath, file)
			if _, err := os.Stat(fullPath); err == nil {
				keyFiles = append(keyFiles, fullPath)
				break
			}
		}
	case "api":
		candidates := []string{
			"main.go", "cmd/main.go",
			"app.js", "server.js", "index.js",
			"app.py", "main.py",
		}
		for _, file := range candidates {
			fullPath := filepath.Join(projectPath, file)
			if _, err := os.Stat(fullPath); err == nil {
				keyFiles = append(keyFiles, fullPath)
				break
			}
		}
	}

	return keyFiles
}

// readKeyFiles reads the content of key files
func (pa *ProjectAnalyzer) readKeyFiles(files []string) map[string]string {
	contents := make(map[string]string)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err == nil {
			// Limit content size to avoid huge prompts
			if len(content) > 5000 {
				content = content[:5000]
			}
			contents[filepath.Base(file)] = string(content)
		}
	}

	return contents
}

// buildAnalysisPrompt builds the analysis prompt for Claude
func (pa *ProjectAnalyzer) buildAnalysisPrompt(projectPath string, structure map[string]int, fileContents map[string]string) string {
	prompt := fmt.Sprintf(`프로젝트 분석을 수행해주세요:

프로젝트 경로: %s

파일 구조:
`, projectPath)

	// Add file structure summary
	for ext, count := range structure {
		if count > 0 && ext != "" {
			prompt += fmt.Sprintf("- %s 파일: %d개\n", ext, count)
		}
	}

	prompt += "\n주요 파일 내용:\n"

	// Add key file contents
	for filename, content := range fileContents {
		prompt += fmt.Sprintf("\n=== %s ===\n%s\n", filename, content)
	}

	prompt += `
다음 사항을 분석해주세요:

1. 코드 품질 문제점
2. 보안 취약점
3. 성능 개선 가능 영역
4. 아키텍처 개선 제안
5. 테스트 커버리지 상태
6. 의존성 업데이트 필요 사항
7. 코드 중복 및 리팩토링 대상

각 문제에 대해 다음 형식으로 응답해주세요:
- 문제 유형 (bug/security/performance/quality)
- 심각도 (critical/high/medium/low)
- 파일 위치
- 문제 설명
- 개선 제안`

	return prompt
}

// parseAnalysisResults parses the analysis results from Claude
func (pa *ProjectAnalyzer) parseAnalysisResults(output string, info *ProjectInfo) {
	// Simple parsing - in production, use more sophisticated parsing
	lines := strings.Split(output, "\n")

	var currentIssue *Issue
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "- 문제 유형:") || strings.HasPrefix(line, "- Type:") {
			if currentIssue != nil && currentIssue.Description != "" {
				info.Issues = append(info.Issues, *currentIssue)
			}
			currentIssue = &Issue{}
			currentIssue.Type = strings.TrimSpace(strings.TrimPrefix(line, "- 문제 유형:"))
		} else if currentIssue != nil {
			if strings.HasPrefix(line, "- 심각도:") || strings.HasPrefix(line, "- Severity:") {
				currentIssue.Severity = strings.TrimSpace(strings.TrimPrefix(line, "- 심각도:"))
			} else if strings.HasPrefix(line, "- 파일:") || strings.HasPrefix(line, "- File:") {
				currentIssue.File = strings.TrimSpace(strings.TrimPrefix(line, "- 파일:"))
			} else if strings.HasPrefix(line, "- 설명:") || strings.HasPrefix(line, "- Description:") {
				currentIssue.Description = strings.TrimSpace(strings.TrimPrefix(line, "- 설명:"))
			} else if strings.HasPrefix(line, "- 제안:") || strings.HasPrefix(line, "- Suggestion:") {
				currentIssue.Suggestion = strings.TrimSpace(strings.TrimPrefix(line, "- 제안:"))
			}
		}

		// Collect improvements
		if strings.Contains(line, "개선") || strings.Contains(line, "improvement") {
			info.Improvements = append(info.Improvements, line)
		}
	}

	// Add last issue if exists
	if currentIssue != nil && currentIssue.Description != "" {
		info.Issues = append(info.Issues, *currentIssue)
	}
}