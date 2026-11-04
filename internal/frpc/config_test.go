package frpc

import (
	"strings"
	"testing"
)

func TestRenderHTTP(t *testing.T) {
	doc, err := Render(ClientConfig{
		Server: "tunnel.example.com", Token: "tok", Transport: "wss",
		Proxies: []Proxy{
			{Name: "app", Type: "http", LocalIP: "127.0.0.1", LocalPort: "3000", Subdomain: "myapp"},
		},
	}, 7000)
	if err != nil {
		t.Fatal(err)
	}
	s := string(doc)
	for _, want := range []string{
		`serverAddr = "tunnel.example.com"`,
		`serverPort = 7000`,
		`auth.token = "tok"`,
		`transport.protocol = "wss"`,
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
	doc, err := Render(ClientConfig{
		Server: "h:7001",
		Proxies: []Proxy{
			{Name: "ssh", Type: "tcp", LocalIP: "127.0.0.1", LocalPort: "22", RemotePort: "2222"},
		},
	}, 7000)
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

func TestRenderRangePrefix(t *testing.T) {
	doc, err := Render(ClientConfig{
		Server: "h",
		Proxies: []Proxy{
			{Name: "web", Type: "tcp", LocalIP: "127.0.0.1",
				LocalPort: "8000-8010", RemotePort: "9000-9010"},
		},
	}, 7000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), `name = "range:web"`) {
		t.Errorf("range: prefix missing:\n%s", doc)
	}
}

func TestRenderShareAndVisitor(t *testing.T) {
	doc, err := Render(ClientConfig{
		Server: "h",
		Proxies: []Proxy{
			{Name: "priv", Type: "stcp", LocalIP: "127.0.0.1",
				LocalPort: "22", SecretKey: "k"},
		},
		Visitors: []Visitor{
			{Name: "v-priv", Type: "stcp", ServerName: "priv",
				SecretKey: "k", BindAddr: "127.0.0.1", BindPort: 2222},
		},
	}, 7000)
	if err != nil {
		t.Fatal(err)
	}
	s := string(doc)
	for _, want := range []string{
		`type = "stcp"`,
		`secretKey = "k"`,
		`[[visitors]]`,
		`serverName = "priv"`,
		`bindPort = 2222`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestRenderPlugin(t *testing.T) {
	doc, err := Render(ClientConfig{
		Server: "h",
		Proxies: []Proxy{
			{Name: "files", Type: "http", Subdomain: "f",
				Plugin: &Plugin{Type: "static_file", LocalPath: "/tmp/pub",
					HTTPUser: "u", HTTPPassword: "p"}},
		},
	}, 7000)
	if err != nil {
		t.Fatal(err)
	}
	s := string(doc)
	for _, want := range []string{
		`[proxies.plugin]`,
		`type = "static_file"`,
		`localPath = "/tmp/pub"`,
		`httpUser = "u"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestRenderRejects(t *testing.T) {
	if _, err := Render(ClientConfig{Server: "h", Proxies: []Proxy{
		{Name: "x", Type: "http", LocalPort: "80"},
	}}, 7000); err == nil {
		t.Error("http proxy without subdomain should fail")
	}
	if _, err := Render(ClientConfig{Server: "h", Proxies: []Proxy{
		{Name: "x", Type: "tcp", LocalPort: "80"},
	}}, 7000); err == nil {
		t.Error("tcp proxy without remote port should fail")
	}
	if _, err := Render(ClientConfig{Server: "h", Proxies: []Proxy{
		{Name: "x", Type: "stcp", LocalPort: "80"},
	}}, 7000); err == nil {
		t.Error("stcp proxy without secretKey should fail")
	}
}
