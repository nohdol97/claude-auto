# CLAUDE.md - Claude Auto-Deploy CLI 개발 가이드

> 이 문서는 Claude가 Claude Auto-Deploy CLI 프로젝트를 구현할 때 필요한 모든 지침과 컨텍스트를 담고 있습니다.

## 🎯 프로젝트 목표

**핵심 목표**: 사용자의 아이디어를 받아 자동으로 전체 프로젝트를 생성, 개발, 테스트, 배포하는 Go 기반 CLI 도구 구현

## 📋 필수 요구사항

### 1. 기술 스택
- **언어**: Go 1.21+
- **CLI 프레임워크**: Cobra
- **Git 라이브러리**: go-git/v5
- **동시성**: Go routines & channels
- **설정**: Viper
- **로깅**: zerolog

### 2. 핵심 원칙
- ✅ **클린 아키텍처**: 계층 분리, 의존성 역전
- ✅ **클린 코드**: SOLID 원칙, DRY, KISS
- ✅ **작은 커밋**: 기능 단위로 atomic commit
- ✅ **병렬 처리**: 독립적 작업은 동시 실행
- ✅ **자동 복구**: Rate limit 및 오류 자동 처리
- ✅ **완전 자동화**: 승인 후 개입 없이 완료까지 진행

## 🏗️ 프로젝트 구조

```
claude-auto/
├── cmd/claude-auto/          # CLI 진입점
├── internal/                 # 내부 패키지 (외부 노출 X)
│   ├── core/                # 핵심 기능
│   ├── tasks/               # 작업 관리
│   ├── generators/          # 코드/프로젝트 생성
│   ├── git/                 # Git 작업
│   ├── docs/                # 문서화
│   └── testing/             # 테스트/검증
├── pkg/types/               # 공유 타입 정의
├── configs/                 # 설정 파일
└── docs/progress/           # 진행 문서 (한글)
```

## 🔧 구현 세부사항

### Phase 1: 프로젝트 초기화 (Day 1)

```bash
# 1. Go 모듈 초기화
go mod init github.com/nohdol/claude-auto

# 2. 필수 의존성 설치
go get github.com/spf13/cobra@v1.8.0
go get github.com/spf13/viper@v1.18.2
go get github.com/go-git/go-git/v5@v5.11.0
go get github.com/rs/zerolog@v1.31.0
go get github.com/fatih/color@v1.16.0
go get github.com/briandowns/spinner@v1.23.0
go get golang.org/x/sync/errgroup@latest
go get github.com/stretchr/testify@v1.8.4

# 3. Makefile 생성
```

**Makefile 내용**:
```makefile
.PHONY: build test run clean install

build:
	go build -o bin/claude-auto cmd/claude-auto/main.go

test:
	go test -v -cover ./...

run:
	go run cmd/claude-auto/main.go

clean:
	rm -rf bin/

install:
	go install ./cmd/claude-auto
```

### Phase 2: Core 모듈 구현

#### 2.1 Claude Executor (`internal/core/claude_executor.go`)

```go
package core

import (
    "bytes"
    "context"
    "os/exec"
    "sync"
    "time"
)

type ClaudeExecutor struct {
    rateLimiter     *RateLimiter
    sessionManager  *SessionManager
    dangerousMode   bool
    maxRetries      int
    mu              sync.Mutex
    activeProcesses map[string]*exec.Cmd
}

// 핵심 메서드
// - Execute(ctx, prompt, options): Claude 실행
// - ExecuteWithRole(ctx, prompt, role): 역할 지정 실행
// - HandleRateLimit(err): Rate limit 처리
// - Cleanup(): 리소스 정리
```

**구현 시 주의사항**:
- `--dangerously-skip-permissions` 플래그 항상 사용
- Rate limit 감지 시 자동 대기 및 재시도
- 컨텍스트 취소 처리
- 프로세스 추적 및 정리

#### 2.2 Rate Limiter (`internal/core/rate_limiter.go`)

```go
type RateLimiter struct {
    mu           sync.RWMutex
    limited      bool
    retryAfter   time.Time
    requests     []time.Time
    maxRequests  int           // 기본값: 10
    window       time.Duration // 기본값: 1분
}

// Rate limit 패턴 감지
func isRateLimitError(output string) bool {
    patterns := []string{
        "rate limit",
        "too many requests",
        "please wait",
        "retry after",
    }
    // 패턴 매칭 로직
}
```

### Phase 3: Task 시스템 구현

#### 3.1 Task 정의 (`pkg/types/types.go`)

```go
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

type TaskType string
const (
    TaskTypeFrontend      TaskType = "frontend"
    TaskTypeBackend       TaskType = "backend"
    TaskTypeDatabase      TaskType = "database"
    TaskTypeTesting       TaskType = "testing"
    TaskTypeDocumentation TaskType = "documentation"
    TaskTypeDevOps        TaskType = "devops"
)
```

#### 3.2 병렬 실행 (`internal/tasks/parallel_executor.go`)

```go
func (pe *ParallelExecutor) ExecuteTasks(tasks []*Task) (*ExecutionReport, error) {
    // 1. 의존성 그래프 생성
    graph := buildDependencyGraph(tasks)

    // 2. Topological sort로 배치 생성
    batches := topologicalSort(graph)

    // 3. 각 배치를 병렬로 실행
    for _, batch := range batches {
        results := pe.executeBatch(batch)
        // 결과 처리
    }
}
```

**Worker Pool 패턴**:
- Frontend Worker: UI 컴포넌트, 스타일링
- Backend Worker: API, 비즈니스 로직
- Database Worker: 스키마, 마이그레이션
- Testing Worker: 테스트 코드, 검증

### Phase 4: 아이디어 처리

#### 4.1 아이디어 프로세서 (`internal/generators/idea_processor.go`)

```go
func (ip *IdeaProcessor) ProcessIdea(idea string) (*ProcessedIdea, error) {
    // 1단계: 아이디어 구체화 프롬프트
    refinementPrompt := fmt.Sprintf(`
당신은 소프트웨어 아키텍트입니다. 다음 아이디어를 구체적인 프로젝트로 변환해주세요:
"%s"

다음 JSON 형식으로 응답해주세요:
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
    "features": ["기능1", "기능2", ...],
    "apis": [
        {"name": "OpenAI", "key": "OPENAI_API_KEY", "required": true}
    ],
    "phases": [
        {"name": "초기 설정", "tasks": [...]},
        {"name": "백엔드 개발", "tasks": [...]},
        {"name": "프론트엔드 개발", "tasks": [...]}
    ]
}`, idea)

    // 2단계: 응답 파싱 및 검증
    // 3단계: 작업 분해
    // 4단계: 의존성 설정
}
```

#### 4.2 작업 분해 전략

```go
func (ip *IdeaProcessor) decomposeTasks(idea ProcessedIdea) []*Task {
    tasks := []*Task{}

    // 1. 프로젝트 초기화 (우선순위 0)
    tasks = append(tasks, &Task{
        ID:       "init-project",
        Type:     TaskTypeDevOps,
        Priority: 0,
        Prompt: `프로젝트 초기화:
- 디렉토리 구조 생성
- package.json 또는 go.mod 생성
- .gitignore, .env.example 생성
- README.md 기본 템플릿`,
    })

    // 2. 데이터베이스 설정 (우선순위 1)
    if idea.HasDatabase {
        tasks = append(tasks, &Task{
            ID:       "setup-database",
            Type:     TaskTypeDatabase,
            Priority: 1,
            Prompt:   "데이터베이스 스키마 설계 및 마이그레이션 파일 생성",
        })
    }

    // 3. 백엔드 API (우선순위 2)
    // 4. 프론트엔드 (우선순위 3)
    // 5. 테스트 (우선순위 4)
    // 6. 문서화 (우선순위 5)

    return tasks
}
```

### Phase 5: Git 자동화

#### 5.1 Git Manager (`internal/git/git_manager.go`)

```go
type GitManager struct {
    repo         *git.Repository
    worktree     *git.Worktree
    commitSize   CommitSize
    author       *object.Signature
}

func (gm *GitManager) SmartCommit(files []string, taskType TaskType) error {
    // 1. 변경사항 분석
    changes := gm.analyzeChanges(files)

    // 2. 커밋 메시지 생성 (Conventional Commits)
    prefix := map[TaskType]string{
        TaskTypeFrontend:  "feat(ui)",
        TaskTypeBackend:   "feat(api)",
        TaskTypeDatabase:  "feat(db)",
        TaskTypeTesting:   "test",
        TaskTypeDevOps:    "ci",
    }[taskType]

    message := fmt.Sprintf("%s: %s", prefix, changes.Summary)

    // 3. 스테이징 및 커밋
    for _, file := range files {
        gm.worktree.Add(file)
    }

    _, err := gm.worktree.Commit(message, &git.CommitOptions{
        Author: gm.author,
    })

    return err
}
```

**커밋 전략**:
- **Atomic**: 하나의 기능 = 하나의 커밋
- **Small**: 작은 변경사항마다 커밋
- **Conventional**: 표준 커밋 메시지 형식

### Phase 6: 문서화 시스템

#### 6.1 문서 생성기 (`internal/docs/doc_generator.go`)

```go
const progressTemplate = `# 프로젝트 진행 상황 보고서

## 📅 날짜: {{.Date}}
## 📊 현재 단계: {{.Phase}}

## ✅ 완료된 작업
{{range .CompletedTasks}}
### {{.Type}}: {{.Title}}
- 시작: {{.StartTime}}
- 완료: {{.EndTime}}
- 소요 시간: {{.Duration}}
- 결과: {{.Result}}
{{end}}

## 🔄 진행 중인 작업
{{range .InProgressTasks}}
### {{.Type}}: {{.Title}}
- 시작: {{.StartTime}}
- 예상 완료: {{.EstimatedTime}}
{{end}}

## 📈 메트릭
- 전체 진행률: {{.Progress}}%
- 코드 라인: {{.LinesOfCode}}
- 커밋 수: {{.CommitCount}}
- 테스트 커버리지: {{.TestCoverage}}%

## 🔑 API 키 상태
{{range .APIKeys}}
- {{.Name}}: {{if .Configured}}✅{{else}}❌{{end}}
{{end}}

## 🚀 다음 단계
{{range .NextSteps}}
1. {{.}}
{{end}}
`
```

### Phase 7: 테스트 및 검증

#### 7.1 빌드 검증기 (`internal/testing/build_validator.go`)

```go
func (bv *BuildValidator) Validate(projectPath string) (*ValidationResult, error) {
    checks := []Check{
        bv.checkDependencies(),   // npm install / go mod download
        bv.runLint(),            // ESLint / golangci-lint
        bv.runBuild(),           // npm run build / go build
        bv.runTests(),           // npm test / go test
        bv.checkSecurity(),      // npm audit / gosec
    }

    for _, check := range checks {
        if !check.Passed {
            // 자동 수정 시도
            bv.attemptAutoFix(check)
        }
    }

    return &ValidationResult{Checks: checks}, nil
}
```

## 📝 프롬프트 템플릿

### 프로젝트 초기화 프롬프트
```
다음 사양으로 프로젝트를 초기화해주세요:
- 프로젝트명: {name}
- 타입: {type}
- 프레임워크: {framework}

생성할 파일:
1. 프로젝트 설정 파일 (package.json, tsconfig.json 등)
2. 디렉토리 구조
3. .gitignore
4. .env.example (필요한 환경 변수 포함)
5. README.md (기본 템플릿)

모든 코드는 TypeScript를 사용하고, 최신 버전을 기준으로 작성해주세요.
```

### 백엔드 API 프롬프트
```
다음 API 엔드포인트를 구현해주세요:
- 프레임워크: {framework}
- 데이터베이스: {database}

구현할 엔드포인트:
{endpoints}

요구사항:
1. RESTful 설계 원칙 준수
2. 입력 검증 및 에러 처리
3. 인증/인가 미들웨어
4. 데이터베이스 연결 및 모델 정의
5. 환경 변수 사용

클린 아키텍처 원칙을 따라 계층을 분리해주세요.
```

### 프론트엔드 컴포넌트 프롬프트
```
다음 UI 컴포넌트를 구현해주세요:
- 프레임워크: {framework}
- 스타일링: {styling}

구현할 컴포넌트:
{components}

요구사항:
1. 재사용 가능한 컴포넌트 설계
2. TypeScript 타입 정의
3. 반응형 디자인
4. 접근성(a11y) 고려
5. 성능 최적화 (메모이제이션 등)

Atomic Design 패턴을 적용해주세요.
```

### 테스트 코드 프롬프트
```
다음 모듈에 대한 테스트 코드를 작성해주세요:
- 테스트 대상: {module}
- 테스트 프레임워크: {framework}

테스트 케이스:
1. 정상 동작 테스트
2. 엣지 케이스 테스트
3. 에러 처리 테스트
4. 통합 테스트

커버리지 목표: 80% 이상
```

## 🔄 워크플로우

### 1. 아이디어 → 프로젝트 계획
```mermaid
graph LR
    A[아이디어 입력] --> B[Claude로 구체화]
    B --> C[기술 스택 결정]
    C --> D[작업 분해]
    D --> E[의존성 매핑]
    E --> F[실행 계획 생성]
```

### 2. 병렬 실행 전략
```
배치 1 (독립 작업):
├── Worker 1: 프로젝트 초기화
├── Worker 2: 데이터베이스 스키마
└── Worker 3: 환경 설정

배치 2 (의존 작업):
├── Worker 1: 백엔드 API
└── Worker 2: 프론트엔드 기본 구조

배치 3 (통합):
├── Worker 1: API 연동
├── Worker 2: 테스트
└── Worker 3: 문서화
```

### 3. Git 커밋 전략
```
[init] 프로젝트 초기 설정
[feat(db)] 데이터베이스 스키마 추가
[feat(api)] 사용자 인증 API 구현
[feat(ui)] 로그인 컴포넌트 추가
[test] 사용자 인증 테스트 추가
[docs] API 문서 업데이트
```

## ⚠️ 중요 규칙

### 1. Claude 실행 규칙
- **항상** `--dangerously-skip-permissions` 플래그 사용
- Rate limit 발생 시 자동 대기 후 재시도
- 세션 간 컨텍스트 유지
- 실패 시 3회까지 자동 재시도

### 2. 코드 품질 규칙
- 모든 코드는 린트 통과 필수
- 테스트 커버리지 60% 이상
- 빌드 성공 확인 후 커밋
- 타입 안정성 보장 (TypeScript/Go)

### 3. Git 규칙
- 커밋 메시지는 Conventional Commits 형식
- 기능 단위로 작은 커밋
- 빌드 실패 코드는 커밋하지 않음
- 매 작업 완료 시 자동 push

### 4. 문서화 규칙
- 모든 진행 상황을 한글로 기록
- 시간별 진행 상황 추적
- 이슈 및 해결 과정 기록
- API 키 및 환경 변수 문서화

## 🎯 성공 기준

1. **자동화 완성도**: 사용자 승인 후 개입 없이 완료
2. **병렬 처리**: 3개 이상 워커 동시 실행
3. **에러 복구**: Rate limit 및 실패 자동 처리
4. **코드 품질**: 린트, 테스트, 빌드 모두 통과
5. **문서화**: 완전한 한글 진행 문서
6. **Git 관리**: 깔끔한 커밋 히스토리

## 🚀 실행 체크리스트

- [ ] Go 1.21+ 설치 확인
- [ ] Claude CLI 설치 및 작동 확인
- [ ] Git 설정 완료
- [ ] GitHub 토큰 설정
- [ ] 프로젝트 구조 생성
- [ ] Core 모듈 구현
- [ ] Task 시스템 구현
- [ ] 아이디어 프로세서 구현
- [ ] Git 자동화 구현
- [ ] 문서화 시스템 구현
- [ ] 테스트 및 검증 시스템 구현
- [ ] CLI 인터페이스 구현
- [ ] 통합 테스트
- [ ] 실제 프로젝트 생성 테스트

## 📚 참고 자료

- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [go-git](https://github.com/go-git/go-git)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Clean Architecture in Go](https://github.com/bxcodec/go-clean-arch)

---

**Note**: 이 문서는 Claude가 프로젝트를 구현할 때 참조해야 할 핵심 가이드입니다. 모든 구현은 이 문서의 지침을 따라야 합니다.