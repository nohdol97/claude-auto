package generators

import (
	"context"
	"fmt"
	"strings"

	"github.com/nohdol/claude-auto/internal/core"
	"github.com/nohdol/claude-auto/internal/git"
	"github.com/nohdol/claude-auto/internal/tasks"
	"github.com/nohdol/claude-auto/pkg/types"
	"github.com/rs/zerolog"
)

// FeatureGenerator generates new features for existing projects
type FeatureGenerator struct {
	claudeExecutor *core.ClaudeExecutor
	taskManager    *tasks.TaskManager
	analyzer       *ProjectAnalyzer
	gitManager     *git.GitManager
	logger         zerolog.Logger
}

// NewFeatureGenerator creates a new feature generator
func NewFeatureGenerator(ce *core.ClaudeExecutor, tm *tasks.TaskManager, logger zerolog.Logger) *FeatureGenerator {
	return &FeatureGenerator{
		claudeExecutor: ce,
		taskManager:    tm,
		analyzer:       NewProjectAnalyzer(ce, logger),
		logger:         logger,
	}
}

// FeatureRequest represents a request to add a new feature
type FeatureRequest struct {
	Name        string
	Description string
	Type        string // api, ui, service, database, etc.
	Components  []string
	ProjectPath string
}

// FeatureResult represents the result of feature generation
type FeatureResult struct {
	FilesCreated  []string
	FilesModified []string
	TestsCreated  []string
	Documentation string
}

// AddFeature adds a new feature to an existing project
func (fg *FeatureGenerator) AddFeature(ctx context.Context, projectPath string, featureName string, featureDescription string) (*FeatureResult, error) {
	fg.logger.Info().
		Str("project", projectPath).
		Str("feature", featureName).
		Msg("Adding new feature to project")

	// First, analyze the existing project
	projectInfo, err := fg.analyzer.AnalyzeProject(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	// Create feature request
	request := &FeatureRequest{
		Name:        featureName,
		Description: featureDescription,
		ProjectPath: projectPath,
	}

	// Determine feature type based on project and description
	request.Type = fg.determineFeatureType(featureDescription, projectInfo)

	// Generate feature plan
	plan, err := fg.generateFeaturePlan(ctx, request, projectInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate feature plan: %w", err)
	}

	// Create tasks for feature implementation
	tasks := fg.createFeatureTasks(request, projectInfo, plan)

	// Execute tasks
	result := &FeatureResult{
		FilesCreated:  []string{},
		FilesModified: []string{},
		TestsCreated:  []string{},
	}

	for _, task := range tasks {
		fg.logger.Info().
			Str("task", task.ID).
			Str("type", string(task.Type)).
			Msg("Executing feature task")

		// Execute task with Claude
		response, err := fg.claudeExecutor.Execute(ctx, task.Prompt, &core.ClaudeOptions{
			Role: fg.getRoleForTaskType(task.Type),
		})
		if err != nil {
			fg.logger.Error().Err(err).Str("task", task.ID).Msg("Task execution failed")
			continue
		}

		// Process response and update result
		fg.processTaskResponse(response.Output, task.Type, result)
	}

	// Generate documentation
	result.Documentation = fg.generateDocumentation(request, result)

	return result, nil
}

// generateFeaturePlan generates a detailed plan for the feature
func (fg *FeatureGenerator) generateFeaturePlan(ctx context.Context, request *FeatureRequest, projectInfo *ProjectInfo) (string, error) {
	prompt := fmt.Sprintf(`프로젝트에 새로운 기능을 추가하려고 합니다.

프로젝트 정보:
- 타입: %s
- 언어: %s
- 프레임워크: %s

추가할 기능:
- 이름: %s
- 설명: %s

다음 사항을 포함한 구현 계획을 수립해주세요:
1. 필요한 컴포넌트/모듈
2. API 엔드포인트 (필요한 경우)
3. 데이터베이스 스키마 변경 (필요한 경우)
4. UI 컴포넌트 (필요한 경우)
5. 테스트 전략
6. 기존 코드와의 통합 포인트

구체적이고 실행 가능한 계획을 JSON 형식으로 제공해주세요.`,
		projectInfo.Type, projectInfo.Language, projectInfo.Framework,
		request.Name, request.Description)

	response, err := fg.claudeExecutor.Execute(ctx, prompt, &core.ClaudeOptions{
		Role:         "software-architect",
		SystemPrompt: "You are an expert software architect who specializes in adding features to existing projects.",
	})

	if err != nil {
		return "", err
	}

	return response.Output, nil
}

// createFeatureTasks creates tasks for implementing the feature
func (fg *FeatureGenerator) createFeatureTasks(request *FeatureRequest, projectInfo *ProjectInfo, plan string) []*types.Task {
	var tasks []*types.Task

	// Determine tasks based on feature type and project type
	switch request.Type {
	case "api":
		tasks = append(tasks, fg.createAPITasks(request, projectInfo, plan)...)
	case "ui":
		tasks = append(tasks, fg.createUITasks(request, projectInfo, plan)...)
	case "fullstack":
		tasks = append(tasks, fg.createFullStackTasks(request, projectInfo, plan)...)
	default:
		tasks = append(tasks, fg.createGenericTasks(request, projectInfo, plan)...)
	}

	// Always add test tasks
	tasks = append(tasks, fg.createTestTasks(request, projectInfo)...)

	return tasks
}

// createAPITasks creates tasks for API features
func (fg *FeatureGenerator) createAPITasks(request *FeatureRequest, projectInfo *ProjectInfo, plan string) []*types.Task {
	tasks := []*types.Task{}

	// Create API endpoint task
	endpointPrompt := fmt.Sprintf(`프로젝트에 새로운 API 엔드포인트를 추가합니다.

기능: %s
설명: %s
프레임워크: %s

구현 계획:
%s

다음을 구현해주세요:
1. 라우터/컨트롤러 코드
2. 서비스/비즈니스 로직
3. 데이터 모델 (필요시)
4. 입력 검증
5. 에러 처리
6. 인증/인가 (필요시)

기존 프로젝트 구조와 일치하는 코드를 생성해주세요.`,
		request.Name, request.Description, projectInfo.Framework, plan)

	task := fg.taskManager.CreateTask(
		types.TaskTypeBackend,
		1,
		endpointPrompt,
	)
	tasks = append(tasks, task)

	// Create database task if needed
	if strings.Contains(strings.ToLower(request.Description), "database") ||
		strings.Contains(strings.ToLower(request.Description), "저장") ||
		strings.Contains(strings.ToLower(request.Description), "조회") {

		dbPrompt := fmt.Sprintf(`데이터베이스 스키마를 추가/수정합니다.

기능: %s
데이터베이스: %s

필요한 테이블, 컬럼, 인덱스, 관계를 정의해주세요.
마이그레이션 파일도 생성해주세요.`, request.Name, projectInfo.Framework)

		dbTask := fg.taskManager.CreateTask(
			types.TaskTypeDatabase,
			0,
			dbPrompt,
		)
		tasks = append(tasks, dbTask)
	}

	return tasks
}

// createUITasks creates tasks for UI features
func (fg *FeatureGenerator) createUITasks(request *FeatureRequest, projectInfo *ProjectInfo, plan string) []*types.Task {
	tasks := []*types.Task{}

	// Create component task
	componentPrompt := fmt.Sprintf(`프로젝트에 새로운 UI 컴포넌트를 추가합니다.

기능: %s
설명: %s
프레임워크: %s

구현 계획:
%s

다음을 구현해주세요:
1. React/Vue/Angular 컴포넌트
2. 스타일링 (CSS/SCSS/Styled Components)
3. 상태 관리
4. 이벤트 핸들링
5. API 연동 (필요시)
6. 반응형 디자인
7. 접근성 고려

기존 프로젝트의 컴포넌트 구조와 일치하게 작성해주세요.`,
		request.Name, request.Description, projectInfo.Framework, plan)

	task := fg.taskManager.CreateTask(
		types.TaskTypeFrontend,
		1,
		componentPrompt,
	)
	tasks = append(tasks, task)

	// Create routing task if needed
	if strings.Contains(strings.ToLower(request.Description), "page") ||
		strings.Contains(strings.ToLower(request.Description), "페이지") {

		routePrompt := fmt.Sprintf(`라우팅을 추가합니다.

기능: %s
새로운 페이지/라우트를 추가하고 네비게이션을 업데이트해주세요.`, request.Name)

		routeTask := fg.taskManager.CreateTask(
			types.TaskTypeFrontend,
			2,
			routePrompt,
		)
		tasks = append(tasks, routeTask)
	}

	return tasks
}

// createFullStackTasks creates tasks for full-stack features
func (fg *FeatureGenerator) createFullStackTasks(request *FeatureRequest, projectInfo *ProjectInfo, plan string) []*types.Task {
	tasks := []*types.Task{}

	// Combine API and UI tasks
	tasks = append(tasks, fg.createAPITasks(request, projectInfo, plan)...)
	tasks = append(tasks, fg.createUITasks(request, projectInfo, plan)...)

	// Add integration task
	integrationPrompt := fmt.Sprintf(`프론트엔드와 백엔드를 통합합니다.

기능: %s

다음을 구현해주세요:
1. API 클라이언트 설정
2. 데이터 페칭 로직
3. 상태 관리
4. 에러 처리
5. 로딩 상태
6. 옵티미스틱 업데이트 (필요시)`,
		request.Name)

	integrationTask := fg.taskManager.CreateTask(
		types.TaskTypeFrontend,
		3,
		integrationPrompt,
	)
	tasks = append(tasks, integrationTask)

	return tasks
}

// createGenericTasks creates generic tasks for any feature type
func (fg *FeatureGenerator) createGenericTasks(request *FeatureRequest, projectInfo *ProjectInfo, plan string) []*types.Task {
	tasks := []*types.Task{}

	genericPrompt := fmt.Sprintf(`프로젝트에 새로운 기능을 추가합니다.

기능: %s
설명: %s
프로젝트 타입: %s
언어: %s
프레임워크: %s

구현 계획:
%s

프로젝트의 기존 구조와 패턴을 따라 기능을 구현해주세요.
필요한 모든 파일과 코드를 생성해주세요.`,
		request.Name, request.Description,
		projectInfo.Type, projectInfo.Language, projectInfo.Framework,
		plan)

	task := fg.taskManager.CreateTask(
		types.TaskTypeBackend, // Default to backend
		1,
		genericPrompt,
	)
	tasks = append(tasks, task)

	return tasks
}

// createTestTasks creates test tasks for the feature
func (fg *FeatureGenerator) createTestTasks(request *FeatureRequest, projectInfo *ProjectInfo) []*types.Task {
	tasks := []*types.Task{}

	testPrompt := fmt.Sprintf(`기능에 대한 테스트를 작성합니다.

기능: %s
설명: %s
언어: %s

다음 테스트를 작성해주세요:
1. 단위 테스트
2. 통합 테스트
3. 엣지 케이스 테스트
4. 에러 처리 테스트

프로젝트의 기존 테스트 구조와 프레임워크를 사용해주세요.`,
		request.Name, request.Description, projectInfo.Language)

	testTask := fg.taskManager.CreateTask(
		types.TaskTypeTesting,
		4,
		testPrompt,
	)
	tasks = append(tasks, testTask)

	return tasks
}

// determineFeatureType determines the type of feature based on description
func (fg *FeatureGenerator) determineFeatureType(description string, projectInfo *ProjectInfo) string {
	desc := strings.ToLower(description)

	// Check for API-related keywords
	apiKeywords := []string{"api", "endpoint", "rest", "graphql", "서버", "백엔드", "데이터베이스"}
	for _, keyword := range apiKeywords {
		if strings.Contains(desc, keyword) {
			if projectInfo.Type == "web" {
				return "fullstack"
			}
			return "api"
		}
	}

	// Check for UI-related keywords
	uiKeywords := []string{"ui", "component", "page", "화면", "버튼", "폼", "프론트엔드", "디자인"}
	for _, keyword := range uiKeywords {
		if strings.Contains(desc, keyword) {
			if projectInfo.Type == "api" {
				return "fullstack"
			}
			return "ui"
		}
	}

	// Check for full-stack keywords
	fullstackKeywords := []string{"full", "전체", "통합", "fullstack", "full-stack"}
	for _, keyword := range fullstackKeywords {
		if strings.Contains(desc, keyword) {
			return "fullstack"
		}
	}

	// Default based on project type
	switch projectInfo.Type {
	case "web":
		return "ui"
	case "api":
		return "api"
	default:
		return "generic"
	}
}

// getRoleForTaskType returns the appropriate Claude role for a task type
func (fg *FeatureGenerator) getRoleForTaskType(taskType types.TaskType) string {
	roleMap := map[types.TaskType]string{
		types.TaskTypeFrontend:      "frontend-developer",
		types.TaskTypeBackend:       "backend-developer",
		types.TaskTypeDatabase:      "database-architect",
		types.TaskTypeTesting:       "qa-engineer",
		types.TaskTypeDocumentation: "technical-writer",
	}

	if role, exists := roleMap[taskType]; exists {
		return role
	}
	return "software-developer"
}

// processTaskResponse processes the response from a task execution
func (fg *FeatureGenerator) processTaskResponse(output string, taskType types.TaskType, result *FeatureResult) {
	// Parse the output to identify created/modified files
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for file paths in the output
		if strings.HasPrefix(line, "Created:") || strings.Contains(line, "created") {
			file := strings.TrimSpace(strings.TrimPrefix(line, "Created:"))
			result.FilesCreated = append(result.FilesCreated, file)
		} else if strings.HasPrefix(line, "Modified:") || strings.Contains(line, "modified") {
			file := strings.TrimSpace(strings.TrimPrefix(line, "Modified:"))
			result.FilesModified = append(result.FilesModified, file)
		}

		// Track test files
		if taskType == types.TaskTypeTesting {
			if strings.Contains(line, ".test.") || strings.Contains(line, ".spec.") ||
				strings.Contains(line, "_test.") {
				result.TestsCreated = append(result.TestsCreated, line)
			}
		}
	}
}

// generateDocumentation generates documentation for the new feature
func (fg *FeatureGenerator) generateDocumentation(request *FeatureRequest, result *FeatureResult) string {
	doc := fmt.Sprintf(`# Feature: %s

## Description
%s

## Implementation Details

### Files Created
`, request.Name, request.Description)

	for _, file := range result.FilesCreated {
		doc += fmt.Sprintf("- %s\n", file)
	}

	doc += "\n### Files Modified\n"
	for _, file := range result.FilesModified {
		doc += fmt.Sprintf("- %s\n", file)
	}

	if len(result.TestsCreated) > 0 {
		doc += "\n### Tests\n"
		for _, test := range result.TestsCreated {
			doc += fmt.Sprintf("- %s\n", test)
		}
	}

	doc += fmt.Sprintf(`
## Usage
TODO: Add usage instructions

## API Reference
TODO: Add API documentation if applicable

## Testing
Run tests with the project's test command.

---
*Generated by Claude Auto-Deploy CLI*
`)

	return doc
}

// SetGitManager sets the git manager for the feature generator
func (fg *FeatureGenerator) SetGitManager(gm *git.GitManager) {
	fg.gitManager = gm
}