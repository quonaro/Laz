package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/quonaro/laz/internal/frpc"
)

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("short string mangled: %q", got)
	}
	if got := truncate("hello world", 5); got != "hell…" {
		t.Errorf("got %q, want %q", got, "hell…")
	}
	if runeLen(truncate("héllo wörld", 6)) != 6 {
		t.Error("rune truncation broken")
	}
}

func TestRuneLen(t *testing.T) {
	if runeLen("abc") != 3 || runeLen("héllo") != 5 || runeLen("═") != 1 {
		t.Error("runeLen miscounts multibyte chars")
	}
}

// TestDrawBannerAligned checks that every drawn line has identical rune
// width — the box math must pad title, rows and footer consistently.
func TestDrawBannerAligned(t *testing.T) {
	buf := &bytes.Buffer{}
	drawBanner(buf, "http", [][2]string{
		{"local", "127.0.0.1:3000"},
		{"↓", ""},
		{"public", "https://my-app.loc.rest"},
	}, "server  frp.example.com:7000", "ttl 1h  encrypt")

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 6 {
		t.Fatalf("too few lines:\n%s", buf.String())
	}
	w := runeLen(lines[0])
	for i, l := range lines {
		if runeLen(l) != w {
			t.Errorf("line %d width %d != %d: %q", i, runeLen(l), w, l)
		}
	}
	mustContain(t, buf.String(), "laz tunnel", "http",
		"https://my-app.loc.rest", "frp.example.com:7000", "ttl 1h")
}

func TestPrintBannerKinds(t *testing.T) {
	cases := []struct {
		name  string
		cc    frpc.ClientConfig
		wants []string
	}{
		{"share", frpc.ClientConfig{
			Server: "h:7000",
			Proxies: []frpc.Proxy{{Name: "n", Type: "stcp",
				LocalIP: "127.0.0.1", LocalPort: "22", SecretKey: "k"}},
		}, []string{"stcp", "laz visit n -k k -y stcp"}},
		{"visitor", frpc.ClientConfig{
			Server: "h:7000",
			Visitors: []frpc.Visitor{{Name: "v", Type: "stcp",
				ServerName: "n", SecretKey: "k",
				BindAddr: "127.0.0.1", BindPort: 2222}},
		}, []string{"visit", "127.0.0.1:2222"}},
		{"plugin", frpc.ClientConfig{
			Server: "h:7000",
			Proxies: []frpc.Proxy{{Name: "n", Type: "tcp", RemotePort: "31000",
				Plugin: &frpc.Plugin{Type: "socks5"}}},
		}, []string{"socks5", "h:31000"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			printBanner(buf, tc.cc, "", "")
			mustContain(t, buf.String(), tc.wants...)
		})
	}
}
