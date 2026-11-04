package frpc

import (
	"fmt"
	"net"
	"strings"
)

// Plugin describes a frpc client plugin attached to a proxy.
type Plugin struct {
	Type         string
	LocalPath    string
	StripPrefix  string
	HTTPUser     string
	HTTPPassword string
	Username     string
	Password     string
	LocalAddr    string
	UnixPath     string
	CrtPath      string
	KeyPath      string
}

// Proxy describes one frpc proxy entry.
type Proxy struct {
	Name           string
	Type           string // http, https, tcp, udp, tcpmux, stcp, sudp, xtcp
	LocalIP        string
	LocalPort      string // "3000" or a range "8000-8010" (needs name prefix "range:")
	RemotePort     string // "40000" or a range matching LocalPort
	Subdomain      string // http/https only
	Domain         string // http/https only: custom_domains base
	SecretKey      string // stcp/sudp/xtcp only
	Plugin         *Plugin
	UseEncryption  bool
	UseCompression bool
	BandwidthLimit string
}

// Visitor describes one frpc visitor entry: the access side of a
// private share (stcp/sudp/xtcp).
type Visitor struct {
	Name       string
	Type       string
	ServerName string
	ServerUser string
	SecretKey  string
	BindAddr   string
	BindPort   int
}

// ClientConfig is everything one frpc run needs.
type ClientConfig struct {
	Server    string
	Token     string
	Transport string
	Proxies   []Proxy
	Visitors  []Visitor
}

// Render produces a frpc.toml document for the given proxies and visitors.
func Render(cc ClientConfig, defaultPort int) ([]byte, error) {
	host, port, err := net.SplitHostPort(cc.Server)
	if err != nil {
		host, port = cc.Server, fmt.Sprint(defaultPort)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "serverAddr = %q\n", host)
	fmt.Fprintf(&b, "serverPort = %s\n", port)
	if cc.Token != "" {
		b.WriteString("auth.method = \"token\"\n")
		fmt.Fprintf(&b, "auth.token = %q\n", cc.Token)
	}
	if cc.Transport != "" {
		fmt.Fprintf(&b, "transport.protocol = %q\n", cc.Transport)
	}
	b.WriteString("log.level = \"info\"\n")
	for _, p := range cc.Proxies {
		if err := writeProxy(&b, p); err != nil {
			return nil, err
		}
	}
	for _, v := range cc.Visitors {
		if err := writeVisitor(&b, v); err != nil {
			return nil, err
		}
	}
	return []byte(b.String()), nil
}

func writeProxy(b *strings.Builder, p Proxy) error {
	switch p.Type {
	case "http", "https":
		if p.Subdomain == "" && p.Domain == "" {
			return fmt.Errorf("%s proxy needs a subdomain", p.Type)
		}
	case "tcp", "udp", "tcpmux":
		if p.RemotePort == "" {
			return fmt.Errorf("%s proxy %q needs a remote port", p.Type, p.Name)
		}
	case "stcp", "sudp", "xtcp":
		if p.SecretKey == "" {
			return fmt.Errorf("%s proxy %q needs a secret key", p.Type, p.Name)
		}
	default:
		return fmt.Errorf("unsupported proxy type %q", p.Type)
	}
	if strings.Contains(p.LocalPort, "-") && !strings.HasPrefix(p.Name, "range:") {
		p.Name = "range:" + p.Name
	}

	fmt.Fprintf(b, "\n[[proxies]]\nname = %q\ntype = %q\n", p.Name, p.Type)
	if p.LocalIP != "" {
		fmt.Fprintf(b, "localIP = %q\n", p.LocalIP)
	}
	if p.LocalPort != "" {
		fmt.Fprintf(b, "localPort = %s\n", p.LocalPort)
	}
	switch p.Type {
	case "http", "https":
		if p.Subdomain != "" {
			fmt.Fprintf(b, "subdomain = %q\n", p.Subdomain)
		} else {
			fmt.Fprintf(b, "customDomains = [%q]\n", p.Domain)
		}
	case "tcp", "udp", "tcpmux":
		fmt.Fprintf(b, "remotePort = %s\n", p.RemotePort)
	default:
		fmt.Fprintf(b, "secretKey = %q\n", p.SecretKey)
	}
	if p.UseEncryption {
		b.WriteString("transport.useEncryption = true\n")
	}
	if p.UseCompression {
		b.WriteString("transport.useCompression = true\n")
	}
	if p.BandwidthLimit != "" {
		fmt.Fprintf(b, "transport.bandwidthLimit = %q\n", p.BandwidthLimit)
	}
	if p.Plugin != nil {
		writePlugin(b, p.Plugin)
	}
	return nil
}

func writePlugin(b *strings.Builder, pl *Plugin) {
	b.WriteString("[proxies.plugin]\n")
	fmt.Fprintf(b, "type = %q\n", pl.Type)
	pairs := [][2]string{
		{"localPath", pl.LocalPath},
		{"stripPrefix", pl.StripPrefix},
		{"httpUser", pl.HTTPUser},
		{"httpPassword", pl.HTTPPassword},
		{"username", pl.Username},
		{"password", pl.Password},
		{"localAddr", pl.LocalAddr},
		{"unixPath", pl.UnixPath},
		{"crtPath", pl.CrtPath},
		{"keyPath", pl.KeyPath},
	}
	for _, kv := range pairs {
		if kv[1] != "" {
			fmt.Fprintf(b, "%s = %q\n", kv[0], kv[1])
		}
	}
}

func writeVisitor(b *strings.Builder, v Visitor) error {
	if v.Name == "" || v.Type == "" || v.ServerName == "" || v.SecretKey == "" {
		return fmt.Errorf("visitor needs name, type, serverName and secretKey")
	}
	fmt.Fprintf(b, "\n[[visitors]]\nname = %q\ntype = %q\n", v.Name, v.Type)
	fmt.Fprintf(b, "serverName = %q\nsecretKey = %q\n", v.ServerName, v.SecretKey)
	if v.ServerUser != "" {
		fmt.Fprintf(b, "serverUser = %q\n", v.ServerUser)
	}
	if v.BindAddr != "" {
		fmt.Fprintf(b, "bindAddr = %q\n", v.BindAddr)
	}
	if v.BindPort != 0 {
		fmt.Fprintf(b, "bindPort = %d\n", v.BindPort)
	}
	return nil
}
