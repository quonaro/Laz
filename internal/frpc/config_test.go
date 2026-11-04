package frpc

import (
	"strings"
	"testing"
)

func TestRenderHTTP(t *testing.T) {
	doc, err := Render("tunnel.example.com", "tok", "websocket", 7000, []Proxy{
		{Name: "app", Type: "http", LocalIP: "127.0.0.1", LocalPort: 3000, Subdomain: "myapp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(doc)
	for _, want := range []string{
		`serverAddr = "tunnel.example.com"`,
		`serverPort = 7000`,
		`auth.token = "tok"`,
		`transport.protocol = "websocket"`,
		`type = "http"`,
		`subdomain = "myapp"`,
		`localPort = 3000`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestRenderTCPRemotePort(t *testing.T) {
	doc, err := Render("h:7001", "", "", 7000, []Proxy{
		{Name: "ssh", Type: "tcp", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 2222},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(doc)
	if !strings.Contains(s, `serverPort = 7001`) {
		t.Errorf("explicit server port not honored:\n%s", s)
	}
	if !strings.Contains(s, `remotePort = 2222`) {
		t.Errorf("remotePort missing:\n%s", s)
	}
	if strings.Contains(s, "auth.token") {
		t.Errorf("token emitted while empty:\n%s", s)
	}
}

func TestRenderRejects(t *testing.T) {
	if _, err := Render("h", "", "", 7000, []Proxy{
		{Name: "x", Type: "http", LocalPort: 80},
	}); err == nil {
		t.Error("http proxy without subdomain should fail")
	}
	if _, err := Render("h", "", "", 7000, []Proxy{
		{Name: "x", Type: "tcp", LocalPort: 80},
	}); err == nil {
		t.Error("tcp proxy without remote port should fail")
	}
}
