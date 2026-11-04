package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolate clears LAZ_* env vars and points both the working directory and
// HOME at fresh temp dirs so no real laz.yaml or ~/.config/laz/config.yaml
// leaks into the test.
func isolate(t *testing.T) string {
	t.Helper()
	for _, k := range []string{"LAZ_SERVER", "LAZ_TOKEN", "LAZ_TRANSPORT"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Chdir(dir)
	return dir
}

func TestLoadFlags(t *testing.T) {
	isolate(t)
	cfg, err := Load("flag.example.com:1234", "tok", "quic")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "flag.example.com:1234" || cfg.Token != "tok" ||
		cfg.Transport != "quic" {
		t.Errorf("flags not applied: %+v", cfg)
	}
	if cfg.DefaultServerPort != 7000 {
		t.Errorf("default port = %d, want 7000", cfg.DefaultServerPort)
	}
}

func TestLoadEnv(t *testing.T) {
	isolate(t)
	t.Setenv("LAZ_SERVER", "env.example.com")
	t.Setenv("LAZ_TOKEN", "envtok")
	t.Setenv("LAZ_TRANSPORT", "wss")
	cfg, err := Load("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "env.example.com" || cfg.Token != "envtok" ||
		cfg.Transport != "wss" {
		t.Errorf("env not applied: %+v", cfg)
	}
}

func TestLoadFile(t *testing.T) {
	dir := isolate(t)
	err := os.WriteFile(filepath.Join(dir, "laz.yaml"), []byte(
		"server: file.example.com\ntoken: filetok\nserver_port: 7001\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "file.example.com" || cfg.Token != "filetok" {
		t.Errorf("file values not applied: %+v", cfg)
	}
	if cfg.DefaultServerPort != 7001 {
		t.Errorf("file server_port = %d, want 7001", cfg.DefaultServerPort)
	}
}

func TestLoadPrecedence(t *testing.T) {
	dir := isolate(t)
	err := os.WriteFile(filepath.Join(dir, "laz.yaml"), []byte(
		"server: file.example.com\ntoken: filetok\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LAZ_SERVER", "env.example.com")

	cfg, err := Load("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "env.example.com" {
		t.Errorf("env should beat file, got %q", cfg.Server)
	}
	if cfg.Token != "filetok" {
		t.Errorf("file token should fill env gap, got %q", cfg.Token)
	}

	cfg, err = Load("flag.example.com", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "flag.example.com" {
		t.Errorf("flag should beat env, got %q", cfg.Server)
	}
}

func TestLoadHomeConfig(t *testing.T) {
	isolate(t)
	home := os.Getenv("HOME")
	dir := filepath.Join(home, ".config", "laz")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(
		"server: home.example.com\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "home.example.com" {
		t.Errorf("home config not applied: %+v", cfg)
	}
}

func TestLoadMissing(t *testing.T) {
	isolate(t)
	if _, err := Load("", "", ""); err == nil {
		t.Fatal("expected error when no server configured")
	}
}

func TestLoadBadFile(t *testing.T) {
	dir := isolate(t)
	err := os.WriteFile(filepath.Join(dir, "laz.yaml"), []byte(":\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load("", "", ""); err == nil {
		t.Fatal("expected parse error")
	}
}
