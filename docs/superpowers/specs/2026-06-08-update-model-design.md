# UpdateModel 추출 및 screenNotification 추가

이슈: #27
날짜: 2026-06-08

## 배경

TUI 모델(`model`)에 자동업데이트 관련 필드/메서드가 직접 포함되어 있다.
formModel(PR #36), searchModel(PR #30)에 이어 updateModel로 분리한다.

현재 `updateDoneMsg` 성공 경로가 `screenError`를 사용하여 "Error: Update successful..."로
표시되는 버그가 있다. 이를 해결하기 위해 `screenNotification`을 추가한다.

## 구조

### updateModel 구조체

```go
// internal/tui/update_model.go
type updateModel struct {
    tag           string           // 사용 가능한 업데이트 태그
    latestRelease *updater.Release // 업데이트 릴리즈 정보
    version       string           // 현재 앱 버전
    checker       UpdateChecker    // 업데이트 체커 인터페이스
}
```

모든 필드 unexported — formModel, searchModel 컨벤션 준수.

### updateModel 메서드

```go
func (u *updateModel) checkUpdate() tea.Cmd
func (u *updateModel) handleConfirmKey(msg tea.KeyMsg) (screen, tea.Cmd)
func (u *updateModel) renderConfirm() string
```

- `checkUpdate()` — 기존 model.checkUpdate() 로직 이동
- `handleConfirmKey()` — y/n/Esc 처리. (screen, tea.Cmd) 반환
- `renderConfirm()` — 기존 renderUpdateConfirm() 로직 이동

### parent model 변경

```go
type model struct {
    // ... 기존 필드 ...
    // 제거: updateTag, latestRelease
    // 이동: Version, Updater → updateModel 내부

    updateModel updateModel  // 새로운 embedded sub-model
    // ...
}
```

`Version`, `Updater` 필드가 updateModel로 이동하므로 parent에서 제거.
`InitialModel(version, updater)` 파라미터는 그대로 유지하며, 내부적으로
`updateModel{version: version, checker: updater}`로 초기화한다.

### screenNotification 추가

```go
const (
    // ... 기존 screens ...
    screenNotification  // 성공/정보 메시지용
)
```

- `updateDoneMsg` 성공 경로 → `screenNotification`
- `updateDoneMsg` 에러 경로 → 기존 `screenError` 유지
- `renderNotification()` 추가 — "Error: " 접두어 없는 일반 메시지 렌더링

### msg 타입 위치

update 관련 msg 타입(`updateAvailableMsg`, `updateCheckFailedMsg`, `updateDoneMsg`)은
기존처럼 `update.go`에 유지. parent Update()에서 switch 라우팅하므로 parent 파일에
있는 게 자연스럽다.

## 파일 변경

| 파일 | 변경 내용 |
|------|----------|
| `internal/tui/update_model.go` | 신규 — updateModel 구조체 + 메서드 |
| `internal/tui/model.go` | updateTag, latestRelease, Version, Updater 제거. updateModel 필드 추가. screenNotification 상수 추가. UpdateChecker 인터페이스 유지(다른 패키지에서 구현체 주입 시 필요). InitialModel에서 updateModel 초기화 |
| `internal/tui/update.go` | checkUpdate 로직 제거(update_model.go로 이동). updateAvailableMsg/updateDoneMsg 핸들러가 m.updateModel 필드 참조 |
| `internal/tui/view.go` | renderUpdateConfirm → m.updateModel.renderConfirm() 위임. renderNotification() 추가. screenNotification case 추가 |
| `internal/tui/update_flow_test.go` | 필드 경로 변경 (m.updateTag → m.updateModel.tag 등). screenError → screenNotification assertion 변경 |
| `internal/tui/update_check_test.go` | m.Version → m.updateModel.version 경로 변경. checkUpdate() 호출 경로 변경 |
| `internal/tui/e2e_test.go` | 필드 경로 변경. 화면 assertion 변경 |

## UpdateChecker 인터페이스

현재 `model.go`에 정의된 `UpdateChecker` 인터페이스는 그대로 유지한다.
이유: 외부 패키지(updater.Client)가 이 인터페이스를 구현하고,
InitialModel에서 구현체를 주입한다. 인터페이스는 consumer(tui) 패키지에
정의하는 Go 관례를 따른다.

updateModel의 `checker` 필드가 이 인터페이스를 참조한다.

## 테스트 영향

| 테스트 파일 | 영향 |
|------------|------|
| `update_flow_test.go` | 필드 경로 변경, screenError → screenNotification assertion |
| `update_check_test.go` | checkUpdate() 호출 경로 변경 (m.checkUpdate() → m.updateModel.checkUpdate()) |
| `e2e_test.go` | 필드 경로 변경, 화면 assertion 변경 |

모든 테스트는 로직 변경 없이 경로만 수정.

## 범위 외

- 실제 DownloadAndInstall() 연결 — 별도 이슈에서 다룸
- renderNotification 스타일 변경 — 기존 renderError 스타일에서 "Error: "만 제거
