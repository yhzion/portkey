package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yhzion/portkey/internal/config"
)

func TestHostStruct(t *testing.T) {
	h := config.Host{Name: "dev", Username: "u", Host: "h", Port: 2222}
	if h.Name != "dev" {
		t.Errorf("Name = %q, want %q", h.Name, "dev")
	}
	if h.Port != 2222 {
		t.Errorf("Port = %d, want %d", h.Port, 2222)
	}
}

func TestHostJSONRoundTrip(t *testing.T) {
	original := config.Host{Name: "dev", Username: "youngho", Host: "192.168.0.10", Port: 22}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var parsed config.Host
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed != original {
		t.Errorf("got %+v, want %+v", parsed, original)
	}
}

func TestHostJSONKeys(t *testing.T) {
	h := config.Host{Name: "x", Username: "u", Host: "h", Port: 22}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"name": "x", "username": "u", "host": "h"}
	for key, want := range expected {
		got, ok := raw[key].(string)
		if !ok || got != want {
			t.Errorf("raw[%q] = %v, want %q", key, raw[key], want)
		}
	}
}

func TestAddHost(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{}}
	cfg.AddHost(config.Host{Name: "first", Username: "u", Host: "h1", Port: 22})

	if len(cfg.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(cfg.Hosts))
	}
	if cfg.Hosts[0].Name != "first" {
		t.Errorf("Hosts[0].Name = %q, want %q", cfg.Hosts[0].Name, "first")
	}

	cfg.AddHost(config.Host{Name: "second", Username: "u", Host: "h2", Port: 2222})
	if len(cfg.Hosts) != 2 {
		t.Fatalf("len(Hosts) = %d, want 2", len(cfg.Hosts))
	}
	if cfg.Hosts[1].Name != "second" {
		t.Errorf("Hosts[1].Name = %q, want %q", cfg.Hosts[1].Name, "second")
	}
}

func TestUpdateHost(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "old", Username: "u", Host: "h", Port: 22},
	}}

	cfg.UpdateHost(0, config.Host{Name: "new", Username: "u2", Host: "h2", Port: 2222})
	if cfg.Hosts[0].Name != "new" {
		t.Errorf("Name = %q, want %q", cfg.Hosts[0].Name, "new")
	}
	if cfg.Hosts[0].Port != 2222 {
		t.Errorf("Port = %d, want %d", cfg.Hosts[0].Port, 2222)
	}
}

func TestUpdateHostOutOfRange(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "only", Username: "u", Host: "h", Port: 22},
	}}

	cfg.UpdateHost(-1, config.Host{Name: "x", Username: "x", Host: "x", Port: 1})
	if cfg.Hosts[0].Name != "only" {
		t.Errorf("Name changed unexpectedly to %q", cfg.Hosts[0].Name)
	}

	cfg.UpdateHost(5, config.Host{Name: "x", Username: "x", Host: "x", Port: 1})
	if cfg.Hosts[0].Name != "only" {
		t.Errorf("Name changed unexpectedly to %q", cfg.Hosts[0].Name)
	}
}

func TestRemoveHost(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "a", Username: "u", Host: "h1", Port: 22},
		{Name: "b", Username: "u", Host: "h2", Port: 22},
		{Name: "c", Username: "u", Host: "h3", Port: 22},
	}}

	cfg.RemoveHost(1)
	if len(cfg.Hosts) != 2 {
		t.Fatalf("len(Hosts) = %d, want 2", len(cfg.Hosts))
	}
	if cfg.Hosts[0].Name != "a" {
		t.Errorf("Hosts[0] = %q, want %q", cfg.Hosts[0].Name, "a")
	}
	if cfg.Hosts[1].Name != "c" {
		t.Errorf("Hosts[1] = %q, want %q", cfg.Hosts[1].Name, "c")
	}
}

func TestRemoveHostOutOfRange(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "only", Username: "u", Host: "h", Port: 22},
	}}

	cfg.RemoveHost(-1)
	if len(cfg.Hosts) != 1 {
		t.Errorf("host removed unexpectedly")
	}

	cfg.RemoveHost(5)
	if len(cfg.Hosts) != 1 {
		t.Errorf("host removed unexpectedly")
	}
}

func TestStoreLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(filepath.Join(dir, "hosts.json"))

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Hosts) != 0 {
		t.Errorf("len(Hosts) = %d, want 0", len(cfg.Hosts))
	}
}

func TestStoreLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.json")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	store := config.NewStore(path)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Hosts) != 0 {
		t.Errorf("len(Hosts) = %d, want 0", len(cfg.Hosts))
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.json")
	store := config.NewStore(path)

	cfg := &config.Config{Hosts: []config.Host{
		{Name: "dev", Username: "youngho", Host: "192.168.0.10", Port: 22},
		{Name: "staging", Username: "ubuntu", Host: "staging.example.com", Port: 2222},
	}}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Hosts) != 2 {
		t.Fatalf("len(Hosts) = %d, want 2", len(loaded.Hosts))
	}
	if loaded.Hosts[0].Name != "dev" {
		t.Errorf("Hosts[0].Name = %q, want %q", loaded.Hosts[0].Name, "dev")
	}
	if loaded.Hosts[0].Username != "youngho" {
		t.Errorf("Hosts[0].Username = %q, want %q", loaded.Hosts[0].Username, "youngho")
	}
	if loaded.Hosts[0].Host != "192.168.0.10" {
		t.Errorf("Hosts[0].Host = %q, want %q", loaded.Hosts[0].Host, "192.168.0.10")
	}
	if loaded.Hosts[0].Port != 22 {
		t.Errorf("Hosts[0].Port = %d, want %d", loaded.Hosts[0].Port, 22)
	}
	if loaded.Hosts[1].Port != 2222 {
		t.Errorf("Hosts[1].Port = %d, want %d", loaded.Hosts[1].Port, 2222)
	}
}

func TestStoreSaveWritesPrivateFileMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.json")
	store := config.NewStore(path)

	if err := os.WriteFile(path, []byte(`{"hosts":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Hosts: []config.Host{
		{Name: "prod", Username: "admin", Host: "10.0.0.1", Port: 22},
	}}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want 600", got)
	}
}

func TestStoreSaveKeepsExistingConfigWhenReplacementCannotBeCreated(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-based save failure cannot be simulated while running as root")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.json")
	store := config.NewStore(path)

	original := &config.Config{Hosts: []config.Host{
		{Name: "prod", Username: "admin", Host: "10.0.0.1", Port: 22},
	}}
	if err := store.Save(original); err != nil {
		t.Fatalf("Save(original) error = %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o700)
	})

	replacement := &config.Config{Hosts: []config.Host{
		{Name: "staging", Username: "ubuntu", Host: "staging.example.com", Port: 2222},
	}}
	if err := store.Save(replacement); err == nil {
		t.Fatal("Save(replacement) should fail when replacement cannot be created")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("config changed after failed save:\ngot:\n%s\nwant:\n%s", after, before)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() after failed save error = %v", err)
	}
	if len(loaded.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(loaded.Hosts))
	}
	if loaded.Hosts[0].Name != "prod" {
		t.Errorf("Hosts[0].Name = %q, want %q", loaded.Hosts[0].Name, "prod")
	}
}

func TestStoreLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.json")
	if err := os.WriteFile(path, []byte("{{invalid json}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := config.NewStore(path)
	_, err := store.Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid JSON")
	}
}

func TestStoreSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "hosts.json")
	store := config.NewStore(path)

	cfg := &config.Config{Hosts: []config.Host{
		{Name: "test", Username: "u", Host: "h", Port: 22},
	}}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Hosts) != 1 {
		t.Errorf("len(Hosts) = %d, want 1", len(loaded.Hosts))
	}
}

func TestStoreLoadRoundTripPreservesData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.json")
	store := config.NewStore(path)

	original := &config.Config{Hosts: []config.Host{
		{Name: "my-server", Username: "admin", Host: "10.0.0.1", Port: 8022},
	}}

	if err := store.Save(original); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	hosts, ok := raw["hosts"].([]interface{})
	if !ok {
		t.Fatal("missing 'hosts' key")
	}
	if len(hosts) != 1 {
		t.Fatalf("raw hosts length = %d, want 1", len(hosts))
	}
	first, ok := hosts[0].(map[string]interface{})
	if !ok {
		t.Fatal("first host is not a map")
	}
	if first["name"] != "my-server" {
		t.Errorf("raw name = %v, want %q", first["name"], "my-server")
	}
}

// --- ValidateName tests ---

func TestValidateNameValid(t *testing.T) {
	names := []string{"my-server", "prod", "feel_so_good", "a", "host-1", "v2_staging", "123"}
	for _, name := range names {
		if err := config.ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateNameRejectsEmpty(t *testing.T) {
	if err := config.ValidateName(""); err == nil {
		t.Error("ValidateName(\"\") should return error")
	}
}

func TestValidateNameRejectsSpaces(t *testing.T) {
	names := []string{"my server", " leading", "trailing ", "multi word name"}
	for _, name := range names {
		if err := config.ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) should reject spaces", name)
		}
	}
}

func TestValidateNameRejectsUppercase(t *testing.T) {
	names := []string{"MyServer", "PROD", "FeelGood", "host_Name"}
	for _, name := range names {
		if err := config.ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) should reject uppercase", name)
		}
	}
}

func TestValidateNameRejectsSpecialChars(t *testing.T) {
	chars := []string{"server.name", "host@name", "host!", "a/b", "a+b", "a b"}
	for _, name := range chars {
		if err := config.ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) should reject special characters", name)
		}
	}
}

// --- IsDuplicateName tests ---

func TestIsDuplicateName(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "prod", Username: "u", Host: "h1", Port: 22},
		{Name: "staging", Username: "u", Host: "h2", Port: 22},
	}}

	if !cfg.IsDuplicateName("prod", -1) {
		t.Error("IsDuplicateName(\"prod\", -1) = false, want true")
	}
	if !cfg.IsDuplicateName("staging", -1) {
		t.Error("IsDuplicateName(\"staging\", -1) = false, want true")
	}
	if cfg.IsDuplicateName("dev", -1) {
		t.Error("IsDuplicateName(\"dev\", -1) = true, want false")
	}
}

func TestIsDuplicateNameExcludeSelf(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "prod", Username: "u", Host: "h1", Port: 22},
		{Name: "staging", Username: "u", Host: "h2", Port: 22},
	}}

	// Index 0 is "prod" — excluding self should return false
	if cfg.IsDuplicateName("prod", 0) {
		t.Error("IsDuplicateName(\"prod\", 0) = true, want false (excluding self)")
	}

	// Index 1 is "staging" — "prod" is still a duplicate at index 0
	if !cfg.IsDuplicateName("prod", 1) {
		t.Error("IsDuplicateName(\"prod\", 1) = false, want true")
	}
}

// --- FindHostByName tests ---

func TestFindHostByNameExactMatch(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
		{Name: "staging", Username: "u", Host: "h2", Port: 22},
	}}

	idx, err := cfg.FindHostByName("production-api")
	if err != nil {
		t.Fatalf("FindHostByName(\"production-api\") error = %v", err)
	}
	if idx != 0 {
		t.Errorf("FindHostByName(\"production-api\") = %d, want 0", idx)
	}
}

func TestFindHostByNameSuffixMatch(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
		{Name: "staging", Username: "u", Host: "h2", Port: 22},
	}}

	idx, err := cfg.FindHostByName("api")
	if err != nil {
		t.Fatalf("FindHostByName(\"api\") error = %v", err)
	}
	if idx != 0 {
		t.Errorf("FindHostByName(\"api\") = %d, want 0", idx)
	}
}

func TestFindHostByNameAmbiguous(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
		{Name: "staging-api", Username: "u", Host: "h2", Port: 22},
	}}

	_, err := cfg.FindHostByName("api")
	if err == nil {
		t.Fatal("FindHostByName(\"api\") should return error for ambiguous match")
	}
}

func TestFindHostByNameNotFound(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
	}}

	_, err := cfg.FindHostByName("staging")
	if err == nil {
		t.Fatal("FindHostByName(\"staging\") should return error when not found")
	}
}

func TestFindHostByNameExactPreferredOverSuffix(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{Name: "api", Username: "u", Host: "h1", Port: 22},
		{Name: "production-api", Username: "u", Host: "h2", Port: 22},
	}}

	idx, err := cfg.FindHostByName("api")
	if err != nil {
		t.Fatalf("FindHostByName(\"api\") error = %v", err)
	}
	// Exact match on "api" should return index 0, not suffix-match on "production-api"
	if idx != 0 {
		t.Errorf("FindHostByName(\"api\") = %d, want 0 (exact match)", idx)
	}
}

// --- MigrateName tests ---

func TestMigrateNameLowercases(t *testing.T) {
	if got := config.MigrateName("ProductionAPI", 0); got != "productionapi" {
		t.Errorf("MigrateName(\"ProductionAPI\") = %q, want %q", got, "productionapi")
	}
}

func TestMigrateNameSpacesToHyphens(t *testing.T) {
	if got := config.MigrateName("Production API", 0); got != "production-api" {
		t.Errorf("MigrateName(\"Production API\") = %q, want %q", got, "production-api")
	}
}

func TestMigrateNameDotsToHyphens(t *testing.T) {
	if got := config.MigrateName("app.local", 0); got != "app-local" {
		t.Errorf("MigrateName(\"app.local\") = %q, want %q", got, "app-local")
	}
}

func TestMigrateNameStripsSpecialChars(t *testing.T) {
	if got := config.MigrateName("my@server!", 0); got != "myserver" {
		t.Errorf("MigrateName(\"my@server!\") = %q, want %q", got, "myserver")
	}
}

func TestMigrateNameEmptyFallsBackToHostIndex(t *testing.T) {
	if got := config.MigrateName("@@@", 3); got != "host-3" {
		t.Errorf("MigrateName(\"@@@\", 3) = %q, want %q", got, "host-3")
	}
}

func TestMigrateNameAlreadyValid(t *testing.T) {
	if got := config.MigrateName("my-server_v2", 0); got != "my-server_v2" {
		t.Errorf("MigrateName(\"my-server_v2\") = %q, want %q", got, "my-server_v2")
	}
}

// --- SortHosts tests ---

func TestSortHostsByLastUsed(t *testing.T) {
	hosts := []config.Host{
		{Name: "old", Username: "u", Host: "h1", Port: 22, LastUsed: ""},
		{Name: "recent", Username: "u", Host: "h2", Port: 22, LastUsed: "2026-06-07T12:00:00Z"},
		{Name: "mid", Username: "u", Host: "h3", Port: 22, LastUsed: "2026-06-06T00:00:00Z"},
	}
	config.SortHosts(hosts)

	if hosts[0].Name != "recent" {
		t.Errorf("hosts[0].Name = %q, want %q", hosts[0].Name, "recent")
	}
	if hosts[1].Name != "mid" {
		t.Errorf("hosts[1].Name = %q, want %q", hosts[1].Name, "mid")
	}
	if hosts[2].Name != "old" {
		t.Errorf("hosts[2].Name = %q, want %q", hosts[2].Name, "old")
	}
}

func TestSortHostsNeverUsedAlphabetical(t *testing.T) {
	hosts := []config.Host{
		{Name: "charlie", Username: "u", Host: "h1", Port: 22, LastUsed: ""},
		{Name: "alpha", Username: "u", Host: "h2", Port: 22, LastUsed: ""},
		{Name: "bravo", Username: "u", Host: "h3", Port: 22, LastUsed: ""},
	}
	config.SortHosts(hosts)

	if hosts[0].Name != "alpha" {
		t.Errorf("hosts[0].Name = %q, want %q", hosts[0].Name, "alpha")
	}
	if hosts[1].Name != "bravo" {
		t.Errorf("hosts[1].Name = %q, want %q", hosts[1].Name, "bravo")
	}
	if hosts[2].Name != "charlie" {
		t.Errorf("hosts[2].Name = %q, want %q", hosts[2].Name, "charlie")
	}
}

func TestSortHostsMixedUsedAndNeverUsed(t *testing.T) {
	hosts := []config.Host{
		{Name: "never-used", Username: "u", Host: "h1", Port: 22, LastUsed: ""},
		{Name: "used", Username: "u", Host: "h2", Port: 22, LastUsed: "2026-06-07T12:00:00Z"},
	}
	config.SortHosts(hosts)

	if hosts[0].Name != "used" {
		t.Errorf("hosts[0].Name = %q, want %q", hosts[0].Name, "used")
	}
	if hosts[1].Name != "never-used" {
		t.Errorf("hosts[1].Name = %q, want %q", hosts[1].Name, "never-used")
	}
}

func TestSortHostsStability(t *testing.T) {
	hosts := []config.Host{
		{Name: "b", Username: "u", Host: "h1", Port: 22, LastUsed: "2026-06-07T12:00:00Z"},
		{Name: "a", Username: "u", Host: "h2", Port: 22, LastUsed: "2026-06-07T12:00:00Z"},
	}
	config.SortHosts(hosts)

	if hosts[0].Name != "b" {
		t.Errorf("hosts[0].Name = %q, want %q (stable)", hosts[0].Name, "b")
	}
	if hosts[1].Name != "a" {
		t.Errorf("hosts[1].Name = %q, want %q (stable)", hosts[1].Name, "a")
	}
}

func TestSortHostsEmpty(t *testing.T) {
	hosts := []config.Host{}
	config.SortHosts(hosts)
	if len(hosts) != 0 {
		t.Errorf("len(hosts) = %d, want 0", len(hosts))
	}
}

func TestHostLastUsedJSONRoundTrip(t *testing.T) {
	original := config.Host{
		Name: "dev", Username: "u", Host: "h", Port: 22,
		LastUsed: "2026-06-07T12:00:00Z",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var parsed config.Host
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.LastUsed != original.LastUsed {
		t.Errorf("LastUsed = %q, want %q", parsed.LastUsed, original.LastUsed)
	}
}

func TestHostLastUsedOmitempty(t *testing.T) {
	h := config.Host{Name: "dev", Username: "u", Host: "h", Port: 22, LastUsed: ""}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["lastUsed"]; ok {
		t.Error("lastUsed should be omitted when empty")
	}
}
