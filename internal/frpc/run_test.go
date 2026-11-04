package frpc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeFrpc writes an executable shell script standing in for frpc.
func fakeFrpc(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "frpc")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunPassesConfig(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "captured.toml")
	bin := fakeFrpc(t, "cp \"$2\" \""+capture+"\"\n")

	cfg := []byte("serverAddr = \"example.com\"\nserverPort = 7000\n")
	if err := Run(context.Background(), bin, cfg, 0); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(cfg) {
		t.Errorf("frpc got different config:\n%s", got)
	}
}

func TestRunFailure(t *testing.T) {
	bin := fakeFrpc(t, "exit 3\n")
	if err := Run(context.Background(), bin, []byte("x"), 0); err == nil {
		t.Fatal("expected frpc failure")
	}
}

func TestRunTTLExpires(t *testing.T) {
	bin := fakeFrpc(t, "exec sleep 30\n")
	start := time.Now()
	err := Run(context.Background(), bin, []byte("x"), 50*time.Millisecond)
	if err != nil {
		t.Fatalf("TTL kill should be silent, got %v", err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("Run did not stop on TTL (%s)", d)
	}
}

func TestRunContextCancel(t *testing.T) {
	bin := fakeFrpc(t, "exec sleep 30\n")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	if err := Run(ctx, bin, []byte("x"), 0); err != nil {
		t.Fatalf("cancel should be silent, got %v", err)
	}
}

func TestLocatePrefersEnv(t *testing.T) {
	bin := fakeFrpc(t, "")
	t.Setenv("LAZ_FRPC", bin)
	if got := Locate(); got != bin {
		t.Errorf("Locate() = %q, want %q", got, bin)
	}
	p, err := Ensure("")
	if err != nil {
		t.Fatal(err)
	}
	if p != bin {
		t.Errorf("Ensure() = %q, want %q", p, bin)
	}
}

func TestLocateSkipsMissingEnv(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-frpc")
	t.Setenv("LAZ_FRPC", missing)
	if got := Locate(); got == missing {
		t.Errorf("Locate() returned nonexistent env path %q", got)
	}
}

func TestManagedPath(t *testing.T) {
	p := managedPath()
	if p == "" || !strings.HasSuffix(p, filepath.Join("laz", "frpc")) {
		t.Errorf("unexpected managed path %q", p)
	}
}
