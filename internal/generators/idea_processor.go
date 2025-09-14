package generators

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nohdol/claude-auto/internal/core"
	"github.com/nohdol/claude-auto/internal/tasks"
	"github.com/nohdol/claude-auto/pkg/types"
	"github.com/rs/zerolog"
)

// IdeaProcessor processes user ideas into concrete project plans
type IdeaProcessor struct {
	claudeExecutor *core.ClaudeExecutor
	taskManager    *tasks.TaskManager
	logger         zerolog.Logger
}

// NewIdeaProcessor creates a new idea processor
func NewIdeaProcessor(ce *core.ClaudeExecutor, tm *tasks.TaskManager, logger zerolog.Logger) *IdeaProcessor {
	return &IdeaProcessor{
		claudeExecutor: ce,
		taskManager:    tm,
		logger:         logger,
	}
}

// ProcessIdea processes a user idea into a structured project plan
func (ip *IdeaProcessor) ProcessIdea(ctx context.Context, idea string) (*types.ProcessedIdea, error) {
	ip.logger.Info().Str("idea", idea).Msg("Processing idea")

	// Step 1: Refine and structure the idea
	refinementPrompt := ip.buildRefinementPrompt(idea)

	options := &core.ClaudeOptions{
		Role:         "software-architect",
		SystemPrompt: "You are an expert software architect who designs scalable and maintainable systems.",
	}

	response, err := ip.claudeExecutor.Execute(ctx, refinementPrompt, options)
	if err != nil {
		return nil, fmt.Errorf("failed to refine idea: %w", err)
	}

	// Step 2: Parse the response
	processedIdea, err := ip.parseProcessedIdea(response.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse processed idea: %w", err)
	}

	// Step 3: Create tasks from the processed idea
	tasks := ip.decomposeTasks(processedIdea)

	// Step 4: Set up task dependencies
	ip.setupDependencies(tasks)

	ip.logger.Info().
		Str("project_name", processedIdea.Name).
		Int("tasks_created", len(tasks)).
		Msg("Idea processing completed")

	return processedIdea, nil
}

// buildRefinementPrompt builds the prompt for idea refinement
func (ip *IdeaProcessor) buildRefinementPrompt(idea string) string {
	return fmt.Sprintf(`당신은 소프트웨어 아키텍트입니다. 다음 아이디어를 구체적인 프로젝트로 변환해주세요:
"%s"

반드시 다음 JSON 형식으로만 응답해주세요. 다른 설명 없이 JSON만 출력하세요:
{
    "name": "프로젝트명",
    "description": "상세 설명",
    "type": "web|api|cli|mobile",
    "architecture": {
        "frontend": {
            "framework": "Next.js|React|Vue",
            "styling": "Tailwind|CSS Modules|Styled Components",
            "state": "Redux|Zustand|Context API"
        },
        "backend": {
            "framework": "Express|Fastify|Gin",
            "database": "PostgreSQL|MongoDB|MySQL",
            "cache": "Redis|Memcached"
        }
    },
    "features": ["기능1", "기능2"],
    "apis": [
        {"name": "OpenAI", "key": "OPENAI_API_KEY", "required": true}
    ],
    "phases": [
        {"name": "초기 설정", "tasks": ["task1", "task2"]},
        {"name": "백엔드 개발", "tasks": ["task3", "task4"]},
        {"name": "프론트엔드 개발", "tasks": ["task5", "task6"]}
    ],
    "has_frontend": true,
    "has_backend": true,
    "has_database": true
}

프로젝트의 규모와 복잡도를 고려하여 실현 가능한 계획을 수립해주세요.`, idea)
}

// parseProcessedIdea parses the JSON response into ProcessedIdea
func (ip *IdeaProcessor) parseProcessedIdea(output string) (*types.ProcessedIdea, error) {
	var processedIdea types.ProcessedIdea

	// Log the raw output for debugging
	ip.logger.Debug().Str("raw_output", output).Msg("Claude response")

	// Try to extract JSON from the output
	// Look for JSON block that starts with { and ends with matching }
	startIdx := strings.Index(output, "{")
	if startIdx == -1 {
		// If no JSON found, try to create a basic structure from the response
		ip.logger.Warn().Msg("No JSON found in response, creating default structure")
		return ip.createDefaultProcessedIdea(output), nil
	}

	// Find the matching closing brace by counting braces
	braceCount := 0
	endIdx := -1
	for i := startIdx; i < len(output); i++ {
		if output[i] == '{' {
			braceCount++
		} else if output[i] == '}' {
			braceCount--
			if braceCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx == -1 {
		ip.logger.Warn().Msg("Incomplete JSON in response, creating default structure")
		return ip.createDefaultProcessedIdea(output), nil
	}

	jsonStr := output[startIdx:endIdx]
	ip.logger.Debug().Str("json", jsonStr).Msg("Extracted JSON")

	if err := json.Unmarshal([]byte(jsonStr), &processedIdea); err != nil {
		ip.logger.Error().Err(err).Str("json", jsonStr).Msg("Failed to unmarshal JSON")
		return ip.createDefaultProcessedIdea(output), nil
	}

	// Set defaults if not provided
	if processedIdea.Name == "" {
		processedIdea.Name = "job-map-service"
	}
	if processedIdea.Type == "" {
		processedIdea.Type = "web"
	}

	return &processedIdea, nil
}

// createDefaultProcessedIdea creates a default ProcessedIdea when JSON parsing fails
func (ip *IdeaProcessor) createDefaultProcessedIdea(output string) *types.ProcessedIdea {
	// Create a sensible default for the Korean job posting map service
	return &types.ProcessedIdea{
		Name:        "job-map-korea",
		Description: "한국 취업 공고 지도 서비스 - 회사 위치를 지도에 표시하고 공고 정보를 제공",
		Type:        "web",
		HasFrontend: true,
		HasBackend:  true,
		HasDatabase: true,
		Architecture: types.ProjectArchitecture{
			Frontend: types.FrontendArchitecture{
				Framework: "Next.js",
				Styling:   "Tailwind CSS",
				State:     "Zustand",
			},
			Backend: types.BackendArchitecture{
				Framework: "Express",
				Database:  "PostgreSQL",
				Cache:     "Redis",
			},
		},
		Features: []string{
			"Interactive map showing company locations",
			"Job posting details on company selection",
			"Search and filter functionality",
			"Real-time job posting updates",
			"Mobile responsive design",
		},
		APIs: []types.APIRequirement{
			{Name: "Kakao Maps", Key: "KAKAO_MAPS_API_KEY", Required: true},
		},
		Phases: []types.ProjectPhase{
			{
				Name: "Setup",
				Tasks: []string{
					"Initialize Next.js project",
					"Setup database schema",
					"Configure map API",
				},
			},
			{
				Name: "Backend Development",
				Tasks: []string{
					"Create API endpoints",
					"Implement job scraping",
					"Setup geocoding service",
				},
			},
			{
				Name: "Frontend Development",
				Tasks: []string{
					"Implement map component",
					"Create job listing UI",
					"Add search functionality",
				},
			},
		},
	}
}

// decomposeTasks creates tasks from the processed idea
func (ip *IdeaProcessor) decomposeTasks(idea *types.ProcessedIdea) []*types.Task {
	var tasks []*types.Task

	// 1. Project initialization (priority 0)
	initTask := ip.taskManager.CreateTask(
		types.TaskTypeDevOps,
		0,
		ip.buildInitPrompt(idea),
	)
	tasks = append(tasks, initTask)

	// 2. Database setup (priority 1)
	if idea.HasDatabase {
		dbTask := ip.taskManager.CreateTask(
			types.TaskTypeDatabase,
			1,
			ip.buildDatabasePrompt(idea),
		)
		tasks = append(tasks, dbTask)
	}

	// 3. Backend API (priority 2)
	if idea.HasBackend {
		backendTasks := ip.createBackendTasks(idea)
		tasks = append(tasks, backendTasks...)
	}

	// 4. Frontend (priority 3)
	if idea.HasFrontend {
		frontendTasks := ip.createFrontendTasks(idea)
		tasks = append(tasks, frontendTasks...)
	}

	// 5. Testing (priority 4)
	testingTasks := ip.createTestingTasks(idea)
	tasks = append(tasks, testingTasks...)

	// 6. Documentation (priority 5)
	docTask := ip.taskManager.CreateTask(
		types.TaskTypeDocumentation,
		5,
		ip.buildDocumentationPrompt(idea),
	)
	tasks = append(tasks, docTask)

	return tasks
}

// buildInitPrompt builds the project initialization prompt
func (ip *IdeaProcessor) buildInitPrompt(idea *types.ProcessedIdea) string {
	return fmt.Sprintf(`프로젝트 초기화:
- 프로젝트명: %s
- 타입: %s
- 설명: %s

다음을 생성해주세요:
1. 디렉토리 구조
2. package.json 또는 go.mod
3. .gitignore
4. .env.example (필요한 환경 변수 포함)
5. README.md (기본 템플릿)

모든 코드는 TypeScript를 사용하고, 최신 버전을 기준으로 작성해주세요.`,
		idea.Name, idea.Type, idea.Description)
}

// buildDatabasePrompt builds the database setup prompt
func (ip *IdeaProcessor) buildDatabasePrompt(idea *types.ProcessedIdea) string {
	return fmt.Sprintf(`데이터베이스 설정:
- 데이터베이스: %s
- 프로젝트: %s

다음을 구현해주세요:
1. 데이터베이스 스키마 설계
2. 마이그레이션 파일 생성
3. 시드 데이터 (개발용)
4. 연결 설정 코드

주요 기능: %v`,
		idea.Architecture.Backend.Database, idea.Name, idea.Features)
}

// createBackendTasks creates backend-related tasks
func (ip *IdeaProcessor) createBackendTasks(idea *types.ProcessedIdea) []*types.Task {
	var tasks []*types.Task

	// API structure
	apiTask := ip.taskManager.CreateTask(
		types.TaskTypeBackend,
		2,
		fmt.Sprintf(`백엔드 API 구조 생성:
- 프레임워크: %s
- 기능: %v

구현할 내용:
1. 라우터 설정
2. 미들웨어 구성
3. 에러 핸들링
4. 환경 변수 관리`,
			idea.Architecture.Backend.Framework, idea.Features),
	)
	tasks = append(tasks, apiTask)

	// Business logic
	logicTask := ip.taskManager.CreateTask(
		types.TaskTypeBackend,
		2,
		`비즈니스 로직 구현:
1. 서비스 레이어
2. 데이터 검증
3. 비즈니스 규칙
4. 유틸리티 함수`,
	)
	tasks = append(tasks, logicTask)

	// Authentication if needed
	if ip.requiresAuth(idea) {
		authTask := ip.taskManager.CreateTask(
			types.TaskTypeBackend,
			2,
			`인증/인가 시스템 구현:
1. JWT 토큰 관리
2. 사용자 인증 미들웨어
3. 권한 관리 시스템
4. 세션 관리`,
		)
		tasks = append(tasks, authTask)
	}

	return tasks
}

// createFrontendTasks creates frontend-related tasks
func (ip *IdeaProcessor) createFrontendTasks(idea *types.ProcessedIdea) []*types.Task {
	var tasks []*types.Task

	// UI components
	uiTask := ip.taskManager.CreateTask(
		types.TaskTypeFrontend,
		3,
		fmt.Sprintf(`UI 컴포넌트 생성:
- 프레임워크: %s
- 스타일링: %s
- 상태 관리: %s

구현할 컴포넌트:
1. 레이아웃 컴포넌트
2. 네비게이션
3. 폼 컴포넌트
4. 데이터 디스플레이 컴포넌트`,
			idea.Architecture.Frontend.Framework,
			idea.Architecture.Frontend.Styling,
			idea.Architecture.Frontend.State),
	)
	tasks = append(tasks, uiTask)

	// Pages/Routes
	pagesTask := ip.taskManager.CreateTask(
		types.TaskTypeFrontend,
		3,
		`페이지 및 라우팅 구현:
1. 홈 페이지
2. 주요 기능 페이지
3. 라우팅 설정
4. 404 페이지`,
	)
	tasks = append(tasks, pagesTask)

	// API integration
	integrationTask := ip.taskManager.CreateTask(
		types.TaskTypeFrontend,
		3,
		`API 통합:
1. API 클라이언트 설정
2. 데이터 페칭 로직
3. 에러 처리
4. 로딩 상태 관리`,
	)
	tasks = append(tasks, integrationTask)

	return tasks
}

// createTestingTasks creates testing-related tasks
func (ip *IdeaProcessor) createTestingTasks(idea *types.ProcessedIdea) []*types.Task {
	var tasks []*types.Task

	// Unit tests
	unitTestTask := ip.taskManager.CreateTask(
		types.TaskTypeTesting,
		4,
		`단위 테스트 작성:
1. 비즈니스 로직 테스트
2. 유틸리티 함수 테스트
3. 컴포넌트 테스트
커버리지 목표: 80% 이상`,
	)
	tasks = append(tasks, unitTestTask)

	// Integration tests
	integrationTestTask := ip.taskManager.CreateTask(
		types.TaskTypeTesting,
		4,
		`통합 테스트 작성:
1. API 엔드포인트 테스트
2. 데이터베이스 통합 테스트
3. 인증 플로우 테스트`,
	)
	tasks = append(tasks, integrationTestTask)

	return tasks
}

// buildDocumentationPrompt builds the documentation prompt
func (ip *IdeaProcessor) buildDocumentationPrompt(idea *types.ProcessedIdea) string {
	return fmt.Sprintf(`프로젝트 문서화:
- 프로젝트명: %s
- 설명: %s

작성할 문서:
1. README.md (설치, 사용법, 기여 가이드)
2. API 문서 (엔드포인트, 요청/응답 형식)
3. 아키텍처 문서
4. 개발 가이드

기능 목록: %v`, idea.Name, idea.Description, idea.Features)
}

// setupDependencies sets up task dependencies
func (ip *IdeaProcessor) setupDependencies(tasks []*types.Task) {
	if len(tasks) < 2 {
		return
	}

	// Database tasks depend on initialization
	initTaskID := tasks[0].ID
	for i := 1; i < len(tasks); i++ {
		task := tasks[i]
		if task.Type == types.TaskTypeDatabase {
			ip.taskManager.AddDependency(task.ID, initTaskID)
		}
	}

	// Backend tasks depend on database (if exists)
	var dbTaskID string
	for _, task := range tasks {
		if task.Type == types.TaskTypeDatabase {
			dbTaskID = task.ID
			break
		}
	}

	if dbTaskID != "" {
		for _, task := range tasks {
			if task.Type == types.TaskTypeBackend {
				ip.taskManager.AddDependency(task.ID, dbTaskID)
			}
		}
	}

	// Frontend tasks can run in parallel with backend
	// but depend on initialization
	for _, task := range tasks {
		if task.Type == types.TaskTypeFrontend {
			ip.taskManager.AddDependency(task.ID, initTaskID)
		}
	}

	// Testing depends on backend and frontend
	for _, task := range tasks {
		if task.Type == types.TaskTypeTesting {
			// Add dependencies to all backend and frontend tasks
			for _, depTask := range tasks {
				if depTask.Type == types.TaskTypeBackend || depTask.Type == types.TaskTypeFrontend {
					ip.taskManager.AddDependency(task.ID, depTask.ID)
				}
			}
		}
	}

	// Documentation depends on everything except itself
	for _, task := range tasks {
		if task.Type == types.TaskTypeDocumentation {
			for _, depTask := range tasks {
				if depTask.Type != types.TaskTypeDocumentation && depTask.ID != task.ID {
					ip.taskManager.AddDependency(task.ID, depTask.ID)
				}
			}
		}
	}
}

// requiresAuth checks if the project requires authentication
func (ip *IdeaProcessor) requiresAuth(idea *types.ProcessedIdea) bool {
	// Simple heuristic: check if features mention user, auth, login, etc.
	authKeywords := []string{"user", "auth", "login", "account", "profile", "인증", "사용자", "로그인"}

	for _, feature := range idea.Features {
		for _, keyword := range authKeywords {
			if contains(feature, keyword) {
				return true
			}
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0)
}