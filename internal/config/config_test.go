package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yhzion/portkey/internal/config"
)

func TestHostStruct(t *testing.T) {
	h := config.Host{DisplayName: "dev", Username: "u", Host: "h", Port: 2222}
	if h.DisplayName != "dev" {
		t.Errorf("DisplayName = %q, want %q", h.DisplayName, "dev")
	}
	if h.Port != 2222 {
		t.Errorf("Port = %d, want %d", h.Port, 2222)
	}
}

func TestHostJSONRoundTrip(t *testing.T) {
	original := config.Host{DisplayName: "dev", Username: "youngho", Host: "192.168.0.10", Port: 22}
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
	h := config.Host{DisplayName: "x", Username: "u", Host: "h", Port: 22}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"displayName": "x", "username": "u", "host": "h"}
	for key, want := range expected {
		got, ok := raw[key].(string)
		if !ok || got != want {
			t.Errorf("raw[%q] = %v, want %q", key, raw[key], want)
		}
	}
}

func TestAddHost(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{}}
	cfg.AddHost(config.Host{DisplayName: "first", Username: "u", Host: "h1", Port: 22})

	if len(cfg.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(cfg.Hosts))
	}
	if cfg.Hosts[0].DisplayName != "first" {
		t.Errorf("Hosts[0].DisplayName = %q, want %q", cfg.Hosts[0].DisplayName, "first")
	}

	cfg.AddHost(config.Host{DisplayName: "second", Username: "u", Host: "h2", Port: 2222})
	if len(cfg.Hosts) != 2 {
		t.Fatalf("len(Hosts) = %d, want 2", len(cfg.Hosts))
	}
	if cfg.Hosts[1].DisplayName != "second" {
		t.Errorf("Hosts[1].DisplayName = %q, want %q", cfg.Hosts[1].DisplayName, "second")
	}
}

func TestUpdateHost(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{DisplayName: "old", Username: "u", Host: "h", Port: 22},
	}}

	cfg.UpdateHost(0, config.Host{DisplayName: "new", Username: "u2", Host: "h2", Port: 2222})
	if cfg.Hosts[0].DisplayName != "new" {
		t.Errorf("DisplayName = %q, want %q", cfg.Hosts[0].DisplayName, "new")
	}
	if cfg.Hosts[0].Port != 2222 {
		t.Errorf("Port = %d, want %d", cfg.Hosts[0].Port, 2222)
	}
}

func TestUpdateHostOutOfRange(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{DisplayName: "only", Username: "u", Host: "h", Port: 22},
	}}

	cfg.UpdateHost(-1, config.Host{DisplayName: "x", Username: "x", Host: "x", Port: 1})
	if cfg.Hosts[0].DisplayName != "only" {
		t.Errorf("DisplayName changed unexpectedly to %q", cfg.Hosts[0].DisplayName)
	}

	cfg.UpdateHost(5, config.Host{DisplayName: "x", Username: "x", Host: "x", Port: 1})
	if cfg.Hosts[0].DisplayName != "only" {
		t.Errorf("DisplayName changed unexpectedly to %q", cfg.Hosts[0].DisplayName)
	}
}

func TestRemoveHost(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{DisplayName: "a", Username: "u", Host: "h1", Port: 22},
		{DisplayName: "b", Username: "u", Host: "h2", Port: 22},
		{DisplayName: "c", Username: "u", Host: "h3", Port: 22},
	}}

	cfg.RemoveHost(1)
	if len(cfg.Hosts) != 2 {
		t.Fatalf("len(Hosts) = %d, want 2", len(cfg.Hosts))
	}
	if cfg.Hosts[0].DisplayName != "a" {
		t.Errorf("Hosts[0] = %q, want %q", cfg.Hosts[0].DisplayName, "a")
	}
	if cfg.Hosts[1].DisplayName != "c" {
		t.Errorf("Hosts[1] = %q, want %q", cfg.Hosts[1].DisplayName, "c")
	}
}

func TestRemoveHostOutOfRange(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{
		{DisplayName: "only", Username: "u", Host: "h", Port: 22},
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
		{DisplayName: "dev", Username: "youngho", Host: "192.168.0.10", Port: 22},
		{DisplayName: "staging", Username: "ubuntu", Host: "staging.example.com", Port: 2222},
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
	if loaded.Hosts[0].DisplayName != "dev" {
		t.Errorf("Hosts[0].DisplayName = %q, want %q", loaded.Hosts[0].DisplayName, "dev")
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
		{DisplayName: "test", Username: "u", Host: "h", Port: 22},
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
		{DisplayName: "my-server", Username: "admin", Host: "10.0.0.1", Port: 8022},
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
	if first["displayName"] != "my-server" {
		t.Errorf("raw displayName = %v, want %q", first["displayName"], "my-server")
	}
}
