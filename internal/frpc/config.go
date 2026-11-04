package frpc

import (
	"fmt"
	"net"
	"strings"
)

// Proxy describes one frpc proxy entry.
type Proxy struct {
	Name       string
	Type       string // http, tcp, udp
	LocalIP    string
	LocalPort  int
	RemotePort int    // tcp/udp only
	Subdomain  string // http only
	Domain     string // http only: custom_domains base
}

// Render produces a frpc.toml document for the given proxies.
func Render(server, token, transport string, defaultPort int, proxies []Proxy) ([]byte, error) {
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		host, port = server, fmt.Sprint(defaultPort)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "serverAddr = %q\n", host)
	fmt.Fprintf(&b, "serverPort = %s\n", port)
	if token != "" {
		b.WriteString("auth.method = \"token\"\n")
		fmt.Fprintf(&b, "auth.token = %q\n", token)
	}
	if transport != "" {
		fmt.Fprintf(&b, "transport.protocol = %q\n", transport)
	}
	b.WriteString("log.level = \"info\"\n")
	for _, p := range proxies {
		if err := writeProxy(&b, p); err != nil {
			return nil, err
		}
	}
	return []byte(b.String()), nil
}

func writeProxy(b *strings.Builder, p Proxy) error {
	switch p.Type {
	case "http":
		if p.Subdomain == "" && p.Domain == "" {
			return fmt.Errorf("http proxy needs a subdomain")
		}
	case "tcp", "udp":
		if p.RemotePort == 0 {
			return fmt.Errorf("%s proxy %q needs a remote port", p.Type, p.Name)
		}
	default:
		return fmt.Errorf("unsupported proxy type %q", p.Type)
	}
	fmt.Fprintf(b, "\n[[proxies]]\nname = %q\ntype = %q\n", p.Name, p.Type)
	fmt.Fprintf(b, "localIP = %q\nlocalPort = %d\n", p.LocalIP, p.LocalPort)
	if p.Type == "http" {
		if p.Subdomain != "" {
			fmt.Fprintf(b, "subdomain = %q\n", p.Subdomain)
		} else {
			fmt.Fprintf(b, "customDomains = [%q]\n", p.Domain)
		}
	} else {
		fmt.Fprintf(b, "remotePort = %d\n", p.RemotePort)
	}
	return nil
}
