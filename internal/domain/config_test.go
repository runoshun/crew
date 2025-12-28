package domain

import "testing"

func TestRepoCrewDir(t *testing.T) {
	got := RepoCrewDir("/home/user/project")
	want := "/home/user/project/.git/crew"
	if got != want {
		t.Errorf("RepoCrewDir() = %q, want %q", got, want)
	}
}

func TestRepoConfigPath(t *testing.T) {
	got := RepoConfigPath("/home/user/project")
	want := "/home/user/project/.git/crew/config.toml"
	if got != want {
		t.Errorf("RepoConfigPath() = %q, want %q", got, want)
	}
}

func TestGlobalCrewDir(t *testing.T) {
	got := GlobalCrewDir("/home/user/.config")
	want := "/home/user/.config/crew"
	if got != want {
		t.Errorf("GlobalCrewDir() = %q, want %q", got, want)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	got := GlobalConfigPath("/home/user/.config")
	want := "/home/user/.config/crew/config.toml"
	if got != want {
		t.Errorf("GlobalConfigPath() = %q, want %q", got, want)
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.Log.Level != DefaultLogLevel {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, DefaultLogLevel)
	}
	if cfg.Agents == nil {
		t.Error("Agents should not be nil")
	}
}
