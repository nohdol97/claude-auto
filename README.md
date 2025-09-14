# Claude Auto-Deploy CLI

🚀 AI 기반 자동 프로젝트 생성 및 배포 도구

## 📋 소개

Claude Auto-Deploy CLI는 사용자의 아이디어를 받아 Claude AI를 활용하여 전체 프로젝트를 자동으로 생성, 개발, 테스트, 배포하는 도구입니다.

### ✨ 주요 기능

- 🤖 **AI 기반 자동 개발**: Claude AI를 활용한 지능형 코드 생성
- ⚡ **병렬 처리**: 프론트엔드, 백엔드, 테스트를 동시에 개발
- 📝 **자동 문서화**: 모든 작업 과정을 한글로 상세히 기록
- 🔄 **스마트 Git 관리**: 작은 단위의 의미있는 커밋 자동 생성
- 🚦 **Rate Limit 관리**: API 제한 자동 감지 및 대기
- ✅ **품질 보증**: 자동 테스트, 빌드 검증, 코드 품질 검사

## 📦 설치

### 필요 사항

- Go 1.21+ (선택사항, 소스 빌드 시)
- Claude CLI (`claude` 명령어)
- Git

### 전역 설치 (권장)

```bash
# 저장소 클론
git clone https://github.com/nohdol/claude-auto.git
cd claude-auto

# 설치 스크립트 실행
sudo ./install.sh
```

설치 스크립트는 다음 작업을 수행합니다:
- `claude-auto` 바이너리를 `/usr/local/bin`에 설치
- 설정 파일을 `~/.claude-auto/`에 복사
- 전역에서 `claude-auto` 명령어 사용 가능

### 수동 설치

```bash
# 빌드
make build

# 바이너리를 PATH에 복사
sudo cp bin/claude-auto /usr/local/bin/
sudo chmod +x /usr/local/bin/claude-auto

# 설정 디렉토리 생성
mkdir -p ~/.claude-auto
cp configs/default.yaml ~/.claude-auto/
```

### Go install 사용

```bash
go install github.com/nohdol/claude-auto/cmd/claude-auto@latest
```

## 🚀 사용법

### 새 프로젝트 생성

```bash
# 현재 디렉토리에 프로젝트 생성
claude-auto idea "실시간 채팅 애플리케이션 만들기"

# 프로젝트가 현재 위치에 하위 폴더로 생성됩니다
cd realtime-chat-app
```

### 기존 프로젝트 개선

```bash
# 현재 프로젝트 분석
claude-auto analyze

# 특정 프로젝트 분석
claude-auto analyze /path/to/project

# 자동 개선 (버그 수정, 성능 최적화)
claude-auto improve

# 특정 문제 수정
claude-auto fix . "메모리 누수 문제 해결"

# 코드 리팩토링
claude-auto refactor
```

### 기능 추가

```bash
# 기존 프로젝트에 새 기능 추가
claude-auto add "사용자 인증" "JWT 기반 로그인/로그아웃 기능 구현"

# 예시: API 기능 추가
claude-auto add "결제 시스템" "Stripe API를 활용한 결제 처리 기능"

# 예시: UI 컴포넌트 추가
claude-auto add "대시보드" "관리자용 통계 대시보드 페이지"

# 예시: 풀스택 기능 추가
claude-auto add "실시간 알림" "WebSocket 기반 실시간 알림 시스템"
```

### 고급 옵션

```bash
# 모든 옵션 활용
claude-auto idea "온라인 쇼핑몰" \
  --workers=5              # 병렬 워커 수
  --auto-approve           # 자동 승인 (확인 없이 진행)
  --type=web              # 프로젝트 타입 지정
  --skip-tests            # 테스트 생략
  --verbose               # 상세 로그 출력
```

### 명령어 옵션

| 옵션 | 설명 | 기본값 |
|------|------|--------|
| `--workers, -w` | 병렬 워커 수 | 3 |
| `--auto-approve, -y` | 자동 승인 | false |
| `--type, -t` | 프로젝트 타입 (web/api/cli/mobile) | auto |
| `--skip-tests` | 테스트 생성 생략 | false |
| `--output, -o` | 출력 디렉토리 | ./ (현재 디렉토리) |
| `--verbose, -v` | 상세 출력 | false |
| `--config` | 설정 파일 경로 | ~/.claude-auto/default.yaml |

## ⚙️ 설정

설정 파일은 `~/.claude-auto/default.yaml`에 위치합니다:

```yaml
claude:
  dangerous_mode: true  # --dangerously-skip-permissions 사용
  max_retries: 3
  timeout: 5m

parallel:
  max_workers: 3        # 병렬 워커 수
  task_timeout: 10m

git:
  auto_commit: true
  commit_size: small    # atomic, small, medium
  push_strategy: batch  # immediate, batch, manual
  author_name: Claude Auto
  author_email: claude-auto@example.com

documentation:
  language: ko          # 한글 문서화
  output_dir: ./docs/progress
  generate: true
```

환경 변수로도 설정 가능:
```bash
export CLAUDE_AUTO_PARALLEL_MAX_WORKERS=5
export CLAUDE_AUTO_GIT_AUTHOR_NAME="Your Name"
```

## 📁 생성되는 프로젝트 구조

```
your-project/
├── src/                 # 소스 코드
├── tests/               # 테스트 코드
├── docs/
│   └── progress/        # 진행 상황 문서 (한글)
├── .git/                # Git 저장소
├── .env.example         # 환경 변수 예제
├── README.md            # 프로젝트 문서
└── package.json         # 또는 go.mod 등
```

## 🔄 워크플로우

1. **아이디어 입력** → Claude가 구체적인 프로젝트 계획 수립
2. **기술 스택 결정** → 최적의 프레임워크 및 도구 선택
3. **작업 분해** → 병렬 실행 가능한 작업으로 분할
4. **병렬 개발** → 프론트엔드, 백엔드, DB를 동시 개발
5. **자동 커밋** → 의미 있는 단위로 Git 커밋
6. **테스트 생성** → 단위 테스트 및 통합 테스트 자동 생성
7. **문서화** → 진행 상황 및 API 문서 자동 생성

## 📊 진행 상황 추적

모든 작업 과정은 `docs/progress/` 디렉토리에 자동으로 기록됩니다:

- 완료된 작업
- 진행 중인 작업
- 프로젝트 메트릭
- 발생한 이슈 및 해결 과정

## 🐛 문제 해결

### Claude CLI가 설치되지 않은 경우
```bash
# Claude CLI 설치 필요
# https://claude.ai/cli 참조
```

### Rate Limit 발생 시
- 자동으로 대기 후 재시도됩니다
- 수동으로 대기 시간 조정: `configs/default.yaml`에서 `claude.timeout` 수정

### Go가 설치되지 않은 경우
- 미리 빌드된 바이너리를 다운로드하거나
- Go 설치 후 소스에서 빌드: https://golang.org/dl/

## 🤝 기여하기

기여를 환영합니다! PR을 보내주세요.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 라이선스

MIT License

## 🙏 감사의 말

- Claude AI by Anthropic
- Go 커뮤니티
- 오픈소스 기여자들

---

Made with ❤️ by Claude Auto-Deploy CLI