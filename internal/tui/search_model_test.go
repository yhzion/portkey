package tui

import (
	"reflect"
	"testing"

	"github.com/yhzion/portkey/internal/config"
)

// --- searchModel.indices() ---

func TestSearchModel_Indices_Inactive_ReturnsNil(t *testing.T) {
	s := &searchModel{}
	if got := s.indices(); got != nil {
		t.Errorf("Indices() = %v, want nil when inactive", got)
	}
}

func TestSearchModel_Indices_Active_AllHosts(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
	}
	s := &searchModel{}
	s.activate(hosts)

	got := s.indices()
	want := []int{0, 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Indices() = %v, want %v", got, want)
	}
}

func TestSearchModel_Indices_Active_ReflectsFilter(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
		{Name: "gamma", Username: "u", Host: "h3", Port: 22},
	}
	s := &searchModel{}
	s.activate(hosts)
	s.query = "al"
	s.updateFilter(hosts)

	got := s.indices()
	if len(got) == 0 {
		t.Fatal("expected at least one filtered index for query 'al'")
	}
	for _, idx := range got {
		if idx < 0 || idx >= len(hosts) {
			t.Errorf("Indices() contains out-of-range index %d", idx)
		}
	}
	wantSubstr := false
	for _, idx := range got {
		if idx == 0 {
			wantSubstr = true
		}
	}
	if !wantSubstr {
		t.Errorf("Indices() = %v, want to contain 0 (alpha)", got)
	}
}

func TestSearchModel_Indices_Active_NoMatches(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
	}
	s := &searchModel{}
	s.activate(hosts)
	s.query = "zzzzzz"
	s.updateFilter(hosts)

	got := s.indices()
	if len(got) != 0 {
		t.Errorf("Indices() = %v, want empty slice for no matches", got)
	}
}

func TestSearchModel_Indices_AfterDeactivate_ReturnsNil(t *testing.T) {
	hosts := []config.Host{{Name: "alpha", Username: "u", Host: "h1", Port: 22}}
	s := &searchModel{}
	s.activate(hosts)
	s.deactivate()

	if got := s.indices(); got != nil {
		t.Errorf("Indices() = %v, want nil after deactivate", got)
	}
}

// --- searchModel.matchMap() ---

func TestSearchModel_MatchMap_Inactive_ReturnsNil(t *testing.T) {
	s := &searchModel{}
	if got := s.matchMap(); got != nil {
		t.Errorf("MatchMap() = %v, want nil when inactive", got)
	}
}

func TestSearchModel_MatchMap_Active_MapsEveryHostIndex(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
	}
	s := &searchModel{}
	s.activate(hosts)

	mm := s.matchMap()
	if mm == nil {
		t.Fatal("MatchMap() = nil, want non-nil when active")
	}
	if len(mm) != len(hosts) {
		t.Errorf("len(MatchMap) = %d, want %d", len(mm), len(hosts))
	}
	for i := range hosts {
		if _, ok := mm[i]; !ok {
			t.Errorf("MatchMap missing key %d", i)
		}
	}
}

func TestSearchModel_MatchMap_Active_PointersReferenceFiltered(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
	}
	s := &searchModel{}
	s.activate(hosts)

	mm := s.matchMap()
	if mm[0] == nil {
		t.Fatal("MatchMap[0] is nil")
	}
	if mm[0].hostIndex != 0 {
		t.Errorf("MatchMap[0].hostIndex = %d, want 0", mm[0].hostIndex)
	}
	if mm[0] != &s.filtered[0] {
		t.Errorf("MatchMap[0] = %p, want %p (pointer to s.filtered[0])", mm[0], &s.filtered[0])
	}
}

func TestSearchModel_MatchMap_AfterDeactivate_ReturnsNil(t *testing.T) {
	hosts := []config.Host{{Name: "alpha", Username: "u", Host: "h1", Port: 22}}
	s := &searchModel{}
	s.activate(hosts)
	s.deactivate()

	if got := s.matchMap(); got != nil {
		t.Errorf("MatchMap() = %v, want nil after deactivate", got)
	}
}
