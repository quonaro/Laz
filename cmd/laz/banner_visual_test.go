package main

import (
	"os"
	"testing"

	"github.com/quonaro/laz/internal/frpc"
)

func TestBannerVisual(t *testing.T) {
	printBanner(os.Stdout, frpc.ClientConfig{
		Server: "frp.example.com:7000",
		Proxies: []frpc.Proxy{{
			Name: "my-api", Type: "http",
			LocalIP: "127.0.0.1", LocalPort: "3000",
			Subdomain: "my-api",
		}},
	}, "my-api.loc.rest", "1h")

	printBanner(os.Stdout, frpc.ClientConfig{
		Server: "frp.example.com:7000", Transport: "quic",
		Proxies: []frpc.Proxy{{
			Name: "laz-tcp-5432", Type: "tcp",
			LocalIP: "127.0.0.1", LocalPort: "5432",
			RemotePort: "45231", UseEncryption: true, UseCompression: true,
		}},
	}, "", "")

	printBanner(os.Stdout, frpc.ClientConfig{
		Server: "frp.example.com:7000",
		Proxies: []frpc.Proxy{{
			Name: "laz-x3f9k2", Type: "stcp",
			LocalIP: "127.0.0.1", LocalPort: "22",
			SecretKey: "m4n5o6p7q8r9s0t1",
		}},
	}, "", "")

	printBanner(os.Stdout, frpc.ClientConfig{
		Server: "frp.example.com:7000",
		Visitors: []frpc.Visitor{{
			Name: "laz-visit-laz-x3f9k2", Type: "stcp",
			ServerName: "laz-x3f9k2", SecretKey: "m4n5o6p7q8r9s0t1",
			BindAddr: "127.0.0.1", BindPort: 23456,
		}},
	}, "", "")

	printBanner(os.Stdout, frpc.ClientConfig{
		Server: "frp.example.com:7000",
		Proxies: []frpc.Proxy{{
			Name: "laz-socks", Type: "tcp", RemotePort: "31000",
			Plugin: &frpc.Plugin{Type: "socks5"},
		}},
	}, "", "")
}
