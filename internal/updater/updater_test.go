package updater

import "testing"

func TestParseVersionTag(t *testing.T) {
	tests := []struct {
		input string
		major int
		minor int
		patch int
		ok    bool
	}{
		{"v0.1.0", 0, 1, 0, true},
		{"v1.2.3", 1, 2, 3, true},
		{"v10.20.30", 10, 20, 30, true},
		{"0.1.0", 0, 1, 0, true},
		{"v0.1", 0, 0, 0, false},
		{"not-a-version", 0, 0, 0, false},
		{"", 0, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			maj, min, pat, ok := ParseVersion(tt.input)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && (maj != tt.major || min != tt.minor || pat != tt.patch) {
				t.Errorf("got %d.%d.%d, want %d.%d.%d", maj, min, pat, tt.major, tt.minor, tt.patch)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v0.1.0", "v0.1.1", true},
		{"v0.1.0", "v0.2.0", true},
		{"v0.1.0", "v1.0.0", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.2.0", "v0.1.0", false},
		{"v1.0.0", "v0.9.9", false},
		{"v0.1.0", "not-a-version", false},
		{"dev", "v0.1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			got := IsNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestPickAsset(t *testing.T) {
	assets := []Asset{
		{Name: "portkey_0.1.0_darwin_amd64.tar.gz", URL: "url1"},
		{Name: "portkey_0.1.0_darwin_arm64.tar.gz", URL: "url2"},
		{Name: "portkey_0.1.0_linux_amd64.tar.gz", URL: "url3"},
		{Name: "portkey_0.1.0_linux_arm64.tar.gz", URL: "url4"},
		{Name: "checksums.txt", URL: "url5"},
	}

	tests := []struct {
		goos   string
		goarch string
		name   string
		ok     bool
	}{
		{"darwin", "arm64", "portkey_0.1.0_darwin_arm64.tar.gz", true},
		{"linux", "amd64", "portkey_0.1.0_linux_amd64.tar.gz", true},
		{"windows", "amd64", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"_"+tt.goarch, func(t *testing.T) {
			got, ok := PickAsset(assets, tt.goos, tt.goarch)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
				return
			}
			if ok && got.Name != tt.name {
				t.Errorf("Name = %q, want %q", got.Name, tt.name)
			}
		})
	}
}
