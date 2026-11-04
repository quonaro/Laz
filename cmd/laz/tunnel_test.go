package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/quonaro/lota/engine"
)

// isolateConfig detaches config resolution from the real environment:
// cwd and HOME are fresh temp dirs (no laz.yaml, no user config) and a
// fixed LAZ_SERVER lets handlers render without a real frps.
func isolateConfig(t *testing.T) {
	t.Helper()
	for _, k := range []string{"LAZ_SERVER", "LAZ_TOKEN", "LAZ_TRANSPORT"} {
		t.Setenv(k, "")
	}
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())
	t.Setenv("LAZ_SERVER", "frp.example.com:7000")
}

// renderCtx builds a NativeContext with --render set so handlers stop
// after writing frpc.toml to the returned buffer.
func renderCtx(args map[string]string) (engine.NativeContext, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	if args == nil {
		args = map[string]string{}
	}
	args["render"] = "true"
	return engine.NativeContext{Args: args, Stdout: buf}, buf
}

func mustContain(t *testing.T, s string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Errorf("missing %q in:\n%s", w, s)
		}
	}
}

func TestRunHTTPRender(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{
		"port": "3000", "host": "127.0.0.1", "subdomain": "myapp",
	})
	if err := runHTTP(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(),
		`serverAddr = "frp.example.com"`,
		`type = "http"`,
		`subdomain = "myapp"`,
		"localPort = 3000",
	)
}

func TestRunHTTPRandomSubdomain(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{"port": "3000", "host": "127.0.0.1"})
	if err := runHTTPS(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(), `type = "https"`, `subdomain = "laz-`)
}

func TestRunTCPRender(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{
		"port": "5432", "remote": "15432", "host": "127.0.0.1",
		"encrypt": "true", "compress": "true",
	})
	if err := runTCP(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(),
		`type = "tcp"`,
		"remotePort = 15432",
		"transport.useEncryption = true",
		"transport.useCompression = true",
	)
}

func TestRunTCPRangeNeedsRemote(t *testing.T) {
	isolateConfig(t)
	nctx, _ := renderCtx(map[string]string{"port": "8000-8010"})
	if err := runTCP(context.Background(), nctx); err == nil {
		t.Fatal("expected range-without-remote error")
	}
}

func TestRunShareRender(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{
		"port": "22", "type": "tcp", "key": "sekret", "name": "myssh",
	})
	if err := runShare(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(),
		`name = "myssh"`,
		`type = "stcp"`,
		`secretKey = "sekret"`,
	)
}

func TestRunShareBadType(t *testing.T) {
	isolateConfig(t)
	nctx, _ := renderCtx(map[string]string{"port": "22", "type": "http"})
	if err := runShare(context.Background(), nctx); err == nil {
		t.Fatal("expected --type error")
	}
}

func TestRunVisitRender(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{
		"name": "myssh", "key": "sekret", "type": "stcp",
		"bind": "0.0.0.0:2222",
	})
	if err := runVisit(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(),
		`name = "laz-visit-myssh"`,
		`serverName = "myssh"`,
		`bindAddr = "0.0.0.0"`,
		"bindPort = 2222",
	)
}

func TestRunVisitBadBind(t *testing.T) {
	isolateConfig(t)
	nctx, _ := renderCtx(map[string]string{
		"name": "x", "key": "k", "type": "stcp", "bind": "noport",
	})
	if err := runVisit(context.Background(), nctx); err == nil {
		t.Fatal("expected bind parse error")
	}
}

func TestRunDirRender(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{
		"path": ".", "subdomain": "files", "user": "u", "pass": "p",
	})
	if err := runDir(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(),
		`subdomain = "files"`,
		`type = "static_file"`,
		`httpUser = "u"`,
	)
}

func TestRunSocksRender(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{
		"remote": "31000", "user": "u", "pass": "p",
	})
	if err := runSocks(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(),
		`type = "socks5"`,
		"remotePort = 31000",
		`username = "u"`,
	)
}

func TestRunXTCPRender(t *testing.T) {
	isolateConfig(t)
	nctx, buf := renderCtx(map[string]string{
		"port": "22", "key": "k", "name": "p2p",
	})
	if err := runXTCP(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	mustContain(t, buf.String(),
		`name = "p2p"`,
		`type = "xtcp"`,
		`secretKey = "k"`,
	)
}

func TestShowVersion(t *testing.T) {
	nctx, buf := renderCtx(nil)
	if err := showVersion(context.Background(), nctx); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.HasPrefix(got, "laz ") {
		t.Errorf("unexpected version output %q", got)
	}
}

func TestSplitHostPort(t *testing.T) {
	h, p, err := splitHostPort("127.0.0.1:8080")
	if err != nil || h != "127.0.0.1" || p != "8080" {
		t.Errorf("got %q:%q, %v", h, p, err)
	}
	h, p, err = splitHostPort("::1:80")
	if err != nil || h != "::1" || p != "80" {
		t.Errorf("last-colon split failed: %q:%q, %v", h, p, err)
	}
	if _, _, err = splitHostPort("noport"); err == nil {
		t.Error("expected error for missing colon")
	}
}

func TestIsRange(t *testing.T) {
	if !isRange("8000-8010") {
		t.Error("range not detected")
	}
	if isRange("8080") {
		t.Error("plain port flagged as range")
	}
}

func TestArgOr(t *testing.T) {
	if argOr("", "def") != "def" || argOr("v", "def") != "v" {
		t.Error("argOr fallback broken")
	}
}

func TestRandomID(t *testing.T) {
	id := randomID(16)
	if len(id) != 16 {
		t.Fatalf("len = %d, want 16", len(id))
	}
	for _, c := range id {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789", c) {
			t.Fatalf("bad char %q in %q", c, id)
		}
	}
}

func TestRandInt(t *testing.T) {
	for range 100 {
		if v := randInt(10); v < 0 || v >= 10 {
			t.Fatalf("randInt(10) = %d out of bounds", v)
		}
	}
}
