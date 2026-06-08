# UpdateModel 추출 및 screenNotification 추가 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 자동업데이트 관련 필드/메서드를 `updateModel` 서브모델로 분리하고, 성공 메시지용 `screenNotification` 화면을 추가한다.

**Architecture:** formModel, searchModel과 동일한 패턴을 따른다. 관련 필드를 unexported 구조체로 묶고, 메서드를 이동시키며, parent model이 위임한다. msg 타입은 기존 `update.go`에 유지한다.

**Tech Stack:** Go, Bubble Tea (charmbracelet), huh

---

### Task 1: screenNotification 추가

새 화면 타입과 렌더러를 추가하여 updateDoneMsg 성공 경로의 "Error: " 버그를 수정한다.

**Files:**
- Modify: `internal/tui/model.go:13-22` (screen 상수)
- Modify: `internal/tui/update.go:31-38` (updateDoneMsg 핸들러)
- Modify: `internal/tui/view.go:11-27` (View switch + renderNotification)
- Modify: `internal/tui/view.go:190-197` (renderError 직후에 renderNotification 추가)
- Test: `internal/tui/view_test.go`

- [ ] **Step 1: screenNotification 상수 추가**

`internal/tui/model.go` — screen 상수 블록에 `screenNotification` 추가:

```go
const (
	screenHostList screen = iota
	screenAddHost
	screenEditHost
	screenDeleteConfirm
	screenUpdateConfirm
	screenError
	screenNotification
)
```

- [ ] **Step 2: renderNotification 메서드 추가**

`internal/tui/view.go` — `renderError()` 직후에 추가:

```go
func (m *model) renderNotification() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(accentStyle.Render(m.errMsg))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press any key to return to host list."))
	return b.String()
}
```

- [ ] **Step 3: View switch에 screenNotification case 추가**

`internal/tui/view.go` — `View()`의 switch에 case 추가:

```go
case screenNotification:
	return m.renderNotification()
```

- [ ] **Step 4: updateDoneMsg 성공 경로를 screenNotification으로 변경**

`internal/tui/update.go` — updateDoneMsg 핸들러의 else 분기 변경:

```go
case updateDoneMsg:
	if msg.err != nil {
		m.errMsg = fmt.Sprintf("Update failed: %s", msg.err.Error())
		m.screen = screenError
	} else {
		m.errMsg = "Update successful. Please restart portkey."
		m.screen = screenNotification
	}
	return m, nil
```

- [ ] **Step 5: handleKey switch에 screenNotification case 추가**

`internal/tui/update.go` — `handleKey()`의 switch에 case 추가:

```go
case screenNotification:
	return m.handleNotificationKey(msg)
```

그리고 `handleErrorKey` 직후에 새 핸들러 추가:

```go
func (m *model) handleNotificationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.screen = screenHostList
	return m, nil
}
```

- [ ] **Step 6: renderNotification 테스트 작성**

`internal/tui/view_test.go` — `TestView_Error` 직후에 추가:

```go
func TestView_Notification(t *testing.T) {
	m := newTestModel()
	m.screen = screenNotification
	m.errMsg = "Update successful. Please restart portkey."
	view := m.View()

	if strings.Contains(view, "Error") {
		t.Error("notification screen should NOT show 'Error' prefix")
	}
	if !strings.Contains(view, "Update successful") {
		t.Error("notification screen should show message")
	}
	if !strings.Contains(view, "any key") {
		t.Error("notification screen should show return hint")
	}
}
```

- [ ] **Step 7: 테스트 실행**

Run: `go test ./internal/tui/ -run "TestView_Notification|TestView_Error" -v`
Expected: 모두 PASS

- [ ] **Step 8: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/view.go internal/tui/view_test.go
git commit -m "feat(tui): add screenNotification for update success messages

Replace screenError with screenNotification in updateDoneMsg success
path. This fixes the misleading 'Error: Update successful...' display."
```

---

### Task 2: updateModel 구조체 생성

업데이트 관련 필드와 메서드를 새로운 `updateModel` 서브모델로 분리한다.

**Files:**
- Create: `internal/tui/update_model.go`
- Modify: `internal/tui/model.go:174-200` (model struct)
- Modify: `internal/tui/model.go:260-274` (InitialModel)
- Modify: `internal/tui/model.go:276-295` (Init + checkUpdate)

- [ ] **Step 1: update_model.go 파일 생성**

`internal/tui/update_model.go` — 신규 파일:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/updater"
)

// updateModel encapsulates the auto-update state and logic.
type updateModel struct {
	tag           string           // available update tag (non-empty when update available)
	latestRelease *updater.Release // release info for the available update
	version       string           // current app version
	checker       UpdateChecker    // update checker (nil in tests/dev)
}

// checkUpdate returns a command that checks for a newer release.
func (u *updateModel) checkUpdate() tea.Cmd {
	return func() tea.Msg {
		rel, err := u.checker.CheckLatest()
		if err != nil {
			return updateCheckFailedMsg{}
		}
		if updater.IsNewer(u.version, rel.Tag) {
			return updateAvailableMsg{Tag: rel.Tag, Rel: rel}
		}
		return nil
	}
}

// handleConfirmKey handles y/n/Esc on the update confirmation screen.
// Returns the target screen and an optional command.
func (u *updateModel) handleConfirmKey(msg tea.KeyMsg, keys keyMap) (screen, tea.Cmd) {
	if key.Matches(msg, keys.Escape) {
		return screenHostList, nil
	}
	switch strings.ToLower(msg.String()) {
	case "y":
		return screenHostList, func() tea.Msg {
			return updateDoneMsg{err: nil}
		}
	case "n":
		return screenHostList, nil
	}
	return screenUpdateConfirm, nil
}

// renderConfirm renders the update confirmation prompt.
func (u *updateModel) renderConfirm() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(accentStyle.Render(fmt.Sprintf("✨ New version (%s) detected. Update now?", u.tag)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[y] yes / [n] no"))
	return b.String()
}
```

- [ ] **Step 2: model struct에서 update 필드 제거하고 updateModel 추가**

`internal/tui/model.go` — model struct 변경:

```go
type model struct {
	screen        screen
	config        *config.Config
	selected      int
	editIndex     int // target row for edit/delete modal ops (shared by form + delete)
	errMsg        string
	keys          keyMap
	width         int
	height        int

	// store persists config. Provided via dependency injection.
	store config.Store

	// Add/edit form state
	formModel formModel

	// Search/filter state
	search searchModel

	// Update state
	updateModel updateModel

	// Last-connected tracking
	connectIndex int  // index of host being connected (-1 = none)
	connected    bool // true after connectHost is called
}
```

제거된 필드: `updateTag`, `latestRelease`, `Version`, `Updater`

- [ ] **Step 3: InitialModel에서 updateModel 초기화**

`internal/tui/model.go` — `InitialModel` 변경:

```go
func InitialModel(cfg *config.Config, version string, upd UpdateChecker, store config.Store) tea.Model {
	if client, ok := upd.(*updater.Client); ok && client == nil {
		upd = nil
	}
	m := &model{
		screen: screenHostList,
		config: cfg,
		selected: 0,
		keys:     newKeyMap(),
		store:    store,
		updateModel: updateModel{
			version: version,
			checker: upd,
		},
	}
	return m
}
```

- [ ] **Step 4: Init에서 updateModel로 위임**

`internal/tui/model.go` — `Init` 변경:

```go
func (m *model) Init() tea.Cmd {
	config.SortHosts(m.config.Hosts)
	if m.updateModel.checker != nil && m.updateModel.version != "dev" {
		return m.updateModel.checkUpdate()
	}
	return nil
}
```

- [ ] **Step 5: model.go에서 기존 checkUpdate 메서드 제거**

`internal/tui/model.go` — `checkUpdate()` 메서드(284-295행) 제거. 이 로직은 이제 `updateModel.checkUpdate()`에 있다.

- [ ] **Step 6: import 정리**

`internal/tui/model.go` — `updater` 패키지 import 제거 (더 이상 model.go에서 직접 사용하지 않음. `updater`는 update_model.go에서 import). `huh` import는 formModel 메서드에서 사용하므로 유지.

최종 model.go imports:

```go
import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/yhzion/portkey/internal/config"
)
```

- [ ] **Step 7: 컴파일 확인**

Run: `go build ./...`
Expected: 컴파일 에러 발생 — `m.updateTag`, `m.latestRelease` 등을 참조하는 update.go, view.go, 테스트 파일들에서 에러. 이는 Task 3-5에서 순차적으로 수정한다.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/update_model.go internal/tui/model.go
git commit -m "refactor(tui): create updateModel sub-model struct

Extract update-related fields (tag, latestRelease, version, checker)
from parent model into dedicated updateModel sub-model. Follows the
same pattern as formModel and searchModel."
```

---

### Task 3: update.go 위임 변경

parent의 update.go 핸들러들이 updateModel 필드를 참조하도록 변경한다.

**Files:**
- Modify: `internal/tui/update.go:23-26` (updateAvailableMsg 핸들러)
- Modify: `internal/tui/update.go:62-76` (handleKey switch)
- Modify: `internal/tui/update.go:106-110` (handleHostListKey의 u 키)
- Modify: `internal/tui/update.go:246-263` (handleUpdateConfirmKey)

- [ ] **Step 1: updateAvailableMsg 핸들러 변경**

`internal/tui/update.go:23-26`:

```go
case updateAvailableMsg:
	m.updateModel.tag = msg.Tag
	m.updateModel.latestRelease = msg.Rel
	return m, nil
```

- [ ] **Step 2: handleUpdateConfirmKey를 updateModel에 위임**

`internal/tui/update.go:246-263` — 기존 메서드를 위임으로 교체:

```go
func (m *model) handleUpdateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	nextScreen, cmd := m.updateModel.handleConfirmKey(msg, m.keys)
	m.screen = nextScreen
	return m, cmd
}
```

- [ ] **Step 3: handleHostListKey의 u 키에서 updateModel 필드 참조**

`internal/tui/update.go:106-110`:

```go
case msg.String() == "u":
	if m.updateModel.latestRelease != nil {
		m.screen = screenUpdateConfirm
		return m, nil
	}
```

- [ ] **Step 4: 테스트 실행**

Run: `go test ./internal/tui/ -run "TestHostList_UKey|TestUpdateConfirm|TestUpdateAvailable" -v`
Expected: 컴파일 에러 — 아직 view.go가 기존 필드를 참조함 (Task 4에서 수정)

- [ ] **Step 5: Commit**

```bash
git add internal/tui/update.go
git commit -m "refactor(tui): delegate update handlers to updateModel

Update message handlers and key handlers now reference updateModel
fields instead of parent model fields."
```

---

### Task 4: view.go 위임 변경

parent의 view.go가 updateModel 필드를 참조하도록 변경한다.

**Files:**
- Modify: `internal/tui/view.go:33` (renderHostList의 updateTag)
- Modify: `internal/tui/view.go:181-188` (renderUpdateConfirm)

- [ ] **Step 1: renderHostList에서 updateModel.tag 참조**

`internal/tui/view.go:33`:

```go
if m.updateModel.tag != "" {
	b.WriteString(" ")
	b.WriteString(accentStyle.Render("✨ " + m.updateModel.tag + " available — press u to update"))
}
```

- [ ] **Step 2: renderUpdateConfirm을 updateModel에 위임**

`internal/tui/view.go:181-188`:

```go
func (m *model) renderUpdateConfirm() string {
	return m.updateModel.renderConfirm()
}
```

- [ ] **Step 3: 컴파일 및 전체 테스트 실행**

Run: `go test ./internal/tui/ -v`
Expected: 컴파일 성공. 테스트는 실패함 (테스트 파일이 기존 필드 경로를 참조, Task 5에서 수정)

- [ ] **Step 4: Commit**

```bash
git add internal/tui/view.go
git commit -m "refactor(tui): delegate view rendering to updateModel

renderHostList and renderUpdateConfirm now reference updateModel
fields instead of parent model fields."
```

---

### Task 5: 테스트 파일 필드 경로 업데이트

모든 테스트가 새 updateModel 필드 경로를 참조하도록 변경하고, screenError → screenNotification assertion을 수정한다.

**Files:**
- Modify: `internal/tui/update_flow_test.go`
- Modify: `internal/tui/update_check_test.go`
- Modify: `internal/tui/view_test.go`
- Modify: `internal/tui/e2e_test.go`

- [ ] **Step 1: update_flow_test.go 변경**

모든 `m.latestRelease` → `m.updateModel.latestRelease`
모든 `m.updateTag` → `m.updateModel.tag`

```go
// TestHostList_UKey_WithUpdateAvailable (line 22)
m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}

// TestUpdateConfirm_Y (line 31)
m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}

// TestUpdateConfirm_N (line 40)
m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}

// TestUpdateConfirm_Esc (line 50)
m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}

// TestUpdateAvailableMsg_SetsNotification (line 60)
if m.updateModel.tag != "v0.2.0" {
	t.Errorf("updateModel.tag = %q, want %q", m.updateModel.tag, "v0.2.0")
}
if m.updateModel.latestRelease == nil {
	t.Error("updateModel.latestRelease should be set")
}

// TestUpdateCheckFailedMsg_Silent (line 77)
if m.updateModel.tag != "" {
	t.Error("update check failure should not set update tag")
}
```

- [ ] **Step 2: update_check_test.go 변경**

`m.checkUpdate()` → `m.updateModel.checkUpdate()` (3곳)

```go
// TestCheckUpdate_NewerAvailable (line 26)
msg := m.updateModel.checkUpdate()()

// TestCheckUpdate_UpToDate (line 41)
if msg := m.updateModel.checkUpdate(); msg() != nil {

// TestCheckUpdate_Error (line 50)
if _, ok := m.updateModel.checkUpdate()().(updateCheckFailedMsg); !ok {
```

- [ ] **Step 3: view_test.go 변경**

`m.updateTag` → `m.updateModel.tag` (2곳), `m.latestRelease` → `m.updateModel.latestRelease` (1곳)

```go
// TestView_UpdateNotification (line 132)
m.updateModel.tag = "v0.2.0"

// TestView_UpdateConfirm (line 145)
m.updateModel.tag = "v0.2.0"
m.updateModel.latestRelease = &updater.Release{Tag: "v0.2.0"}
```

- [ ] **Step 4: e2e_test.go 변경**

update 관련 필드 경로 4곳 변경:

```go
// TestE2E_View_HostListWithUpdate (line 1099)
m.updateModel.tag = "v0.2.0"

// TestE2E_View_UpdateConfirm (line 1105)
m.updateModel.tag = "v0.2.0"

// TestE2E_Update_UKeyWithUpdate (line 1230)
m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}

// TestE2E_Edge_UpdateConfirmYesNo (line 1264)
m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}
```

- [ ] **Step 5: 전체 테스트 실행**

Run: `go test ./internal/tui/ -v`
Expected: 모든 테스트 PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/update_flow_test.go internal/tui/update_check_test.go internal/tui/view_test.go internal/tui/e2e_test.go
git commit -m "test(tui): update field paths after updateModel extraction

Update all test files to reference updateModel sub-model fields.
No logic changes — only field path adjustments."
```

---

### Task 6: 최종 검증

모든 변경이 완료된 후 전체 테스트 스위트를 실행하고 빌드를 확인한다.

**Files:**
- All modified files

- [ ] **Step 1: 전체 테스트 실행**

Run: `go test ./... -v`
Expected: 모든 패키지 PASS

- [ ] **Step 2: 빌드 확인**

Run: `go build ./...`
Expected: 에러 없음

- [ ] **Step 3: lint 확인**

Run: `go vet ./...`
Expected: 문제 없음

- [ ] **Step 4: 최종 Commit (필요 시)**

lint/format 수정이 필요한 경우만:

```bash
git add -A
git commit -m "chore: final cleanup after updateModel extraction"
```
