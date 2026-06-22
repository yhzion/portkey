# 호스트 헬스체크 및 온라인 상태 표시

날짜: 2026-06-22

## 배경

호스트 목록에서 각 호스트가 현재 접속 가능한지 한눈에 알 수 없다. 접속을 시도해야만
도달 여부를 알 수 있어, 죽은 호스트를 고를 때 시간을 낭비한다.

각 호스트의 도달 가능 여부를 백그라운드에서 확인하고, 그 결과를 호스트 목록에
**텍스트 색상**으로 표시한다. 접속 가능한 호스트의 이름은 초록색으로 보인다.

## 목표 / 비목표

**목표**
- 시작 시 모든 호스트의 도달 가능 여부를 백그라운드에서 확인
- 결과를 호스트 이름 색상으로 표시 (online=초록 / offline=회색 / unknown=기본색)
- `r` 키로 수동 재확인
- UI를 절대 블로킹하지 않음 (Bubbletea 이벤트 루프 보존)

**비목표 (이번 범위 밖)**
- 주기적 자동 재확인 (`tea.Tick`) — 나중에 옵션으로 추가 가능
- 글리프(`●`) 기반 표시 / 색맹 대체 표기 — 색상 단독으로 시작
- SSH config(`~/.ssh/config`) 파싱으로 별칭 해석
- 응답 시간(latency) 표시

## 메커니즘: TCP dial (ICMP ping 아님)

도달성 판정은 **호스트의 SSH 포트로의 TCP 연결**로 한다.

```go
conn, err := net.DialTimeout("tcp", net.JoinHostPort(host.Host, strconv.Itoa(host.Port)), timeout)
```

- `timeout`: 1.5초
- 성공 → `online`, 닫힘/거부/타임아웃 → `offline`, **DNS 해석 실패 → `unknown`**

ICMP ping을 쓰지 않는 이유:
- ICMP raw socket은 root/`CAP_NET_RAW` 권한 필요 → **Termux 비루트에서 사용 불가**
- TCP dial은 권한 불필요·크로스플랫폼이며, "SSH로 접속 가능한가"라는 **실제 의미**를 측정

### 별칭 호스트 처리

`yhzion@rtx5090`, `feel_so_good@youngho`의 `rtx5090`/`youngho`는 DNS로 해석되지 않는
SSH config 별칭일 수 있다. 이때 TCP dial은 DNS 단계에서 실패한다. 이를 `offline`(빨강/회색)
으로 표시하면 **거짓 정보**가 되므로, DNS 해석 실패는 `unknown`(기본색, 중립)으로 분류한다.

판정 기준:
- `*net.DNSError` (errors.As로 검출) → `unknown`
- 그 외 dial 에러 (refused/timeout 등) → `offline`
- 성공 → `online`

## 구조

### 상태 타입

```go
// internal/tui/health.go
type healthStatus int

const (
    healthUnknown healthStatus = iota // 미확인/확인중/DNS 해석 불가
    healthOnline                      // TCP 연결 성공
    healthOffline                     // 연결 거부/타임아웃
)
```

### 순수 판정 함수 (테스트 용이)

```go
// classifyDialError가 dial 결과 에러를 상태로 변환한다. err == nil → online.
func classifyDialError(err error) healthStatus
```

- `err == nil` → `healthOnline`
- DNS 해석 실패(`var dnsErr *net.DNSError; errors.As(err, &dnsErr)`) → `healthUnknown`
- 그 외 → `healthOffline`

이 함수는 네트워크 없이 단위 테스트한다 (DNSError / 일반 에러 / nil).

### 비동기 실행 (기존 checkUpdate 패턴 미러링)

`updateModel.checkUpdate(ctx) tea.Cmd { return func() tea.Msg {...} }`
(update_model.go) 패턴을 그대로 따른다.

```go
// 호스트 하나를 확인하는 Cmd. 결과를 healthResultMsg로 반환.
func checkHostCmd(name, host string, port int) tea.Cmd {
    return func() tea.Msg {
        err := dialHost(host, port, healthTimeout)
        return healthResultMsg{name: name, status: classifyDialError(err)}
    }
}

type healthResultMsg struct {
    name   string
    status healthStatus
}
```

- 모든 호스트 체크를 `tea.Batch(cmds...)`로 동시 발사
- 각 결과가 독립적으로 도착 → 해당 행만 점진적으로 색이 바뀜 (반응성 ↑)
- 동시 dial 수가 많아질 우려는 현재 호스트 규모(~10)에서 무시 가능. 대규모 대비 동시성
  상한은 비목표로 미룬다.

### 모델 상태

```go
// model에 추가
health map[string]healthStatus // key: host.Name (validator가 유일성 보장)
```

- 초기값: 키 부재 = `healthUnknown` (맵 zero value와 일치하므로 명시 초기화 불필요)
- `healthResultMsg` 수신 시 `m.health[msg.name] = msg.status`

### 트리거

- **시작 시**: `Init()`이 반환하는 Cmd에 `healthCheckAllCmd(m.config.Hosts)`를 `tea.Batch`로 합류
- **수동 재확인**: 호스트 목록 화면에서 `r` 키 → 모든 호스트 재확인 Cmd 재발사. 재확인 중인
  호스트는 즉시 `healthUnknown`으로 되돌려 "확인중"을 표현

### Update 처리

`update.go`의 `Update` switch에 케이스 추가:

```go
case healthResultMsg:
    m.health[msg.name] = msg.status
    return m, nil
```

`r` 키는 `handleHostListKey`에 추가.

## 표시 디자인

호스트 **이름 텍스트의 색**으로 상태를 인코딩한다. 레이아웃(이름 칼럼 폭, 정렬)은
변경하지 않는다 — 색만 입힌다.

| 상태 | 이름 색 | 스타일 상수 |
|------|---------|-------------|
| online | 초록 | `colorPositive` (기존) |
| offline | 회색 dim | `colorDim` (기존) |
| unknown | 기본색 | 현재 `nameStyle` 그대로 |

`view.go`의 `renderHostItem`에서, 이름 셀 렌더 시 상태에 따라 foreground 색을 적용한다.

```go
nameStyled := nameStyle.Width(nameWidth)
switch status {
case healthOnline:
    nameStyled = nameStyled.Foreground(lipgloss.Color(colorPositive))
case healthOffline:
    nameStyled = nameStyled.Foreground(lipgloss.Color(colorDim))
}
// healthUnknown은 기본색 유지
```

### 선택행 / 검색 하이라이트와의 상호작용

- 선택행은 `selectedStyle`(배경색)로 행 전체를 감싼다. 이미 `highlightMatched`가 검색
  매칭을 행 내부 accent 색으로 칠하는 선례가 있어, 이름 세그먼트에 색을 중첩해도 깨지지
  않는다.
- 검색 매칭으로 이름이 하이라이트된 경우, 매칭 글자(accent)가 상태색보다 우선한다 (기존
  `highlightMatched` 동작 유지). 비매칭 글자는 상태색을 따른다. → 별도 처리 없이 자연스럽게
  공존.

## 에러 처리

- dial 실패는 정상 흐름의 일부 (offline/unknown으로 분류) — 에러 화면 띄우지 않음
- 헬스체크는 best-effort: 어떤 이유로든 결과가 안 오면 해당 호스트는 `unknown`으로 남음
  (기본색) → 사용자 경험에 무해

## 테스트 전략

순수/결정적 단위 테스트 우선 (네트워크 의존 회피):

1. `classifyDialError` — nil→online, `*net.DNSError`→unknown, 일반 에러→offline
2. `renderHostItem` 색상 — 각 상태에서 이름 세그먼트에 올바른 색 코드가 적용되는지
   (렌더 문자열에 해당 ANSI 색 포함 여부)
3. `Update`의 `healthResultMsg` 처리 — 맵에 상태가 반영되는지
4. `r` 키 — 재확인 Cmd가 발사되고 상태가 unknown으로 리셋되는지

실제 TCP dial(`dialHost`)은 얇은 래퍼로 두고 단위 테스트 대상에서 제외 (표준 라이브러리
위임). 통합은 기존 e2e 패턴이 있으면 거기에 한 줄 추가 검토.

## 미해결/후속 (비목표)

- 주기적 자동 재확인 (`tea.Tick`, 예: 30초)
- 색맹 대체: `●`/`○` 글리프를 설정 플래그로 추가
- 동시 dial 상한 (대규모 호스트 목록 대비)
- 타이틀 바 집계 표시 ("4/6 online")
