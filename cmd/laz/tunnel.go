package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/quonaro/laz/internal/config"
	"github.com/quonaro/laz/internal/frpc"
	"github.com/quonaro/lota/engine"
)

func runHTTP(ctx context.Context, nctx engine.NativeContext) error {
	return runVhost(ctx, nctx, "http")
}

func runHTTPS(ctx context.Context, nctx engine.NativeContext) error {
	return runVhost(ctx, nctx, "https")
}

func runVhost(ctx context.Context, nctx engine.NativeContext, typ string) error {
	sub := nctx.Args["subdomain"]
	if sub == "" {
		sub = "laz-" + randomID(6)
	}
	p := frpc.Proxy{
		Name: argOr(nctx.Args["name"], sub), Type: typ,
		LocalIP: nctx.Args["host"], LocalPort: nctx.Args["port"],
		Subdomain: sub,
	}
	applyCommon(&p, nctx)
	return run(ctx, nctx, frpc.ClientConfig{Proxies: []frpc.Proxy{p}},
		func() string {
			return fmt.Sprintf("https://%s.loc.rest", p.Subdomain)
		})
}

func runTCP(ctx context.Context, nctx engine.NativeContext) error {
	return runRaw(ctx, nctx, "tcp")
}

func runUDP(ctx context.Context, nctx engine.NativeContext) error {
	return runRaw(ctx, nctx, "udp")
}

func runRaw(ctx context.Context, nctx engine.NativeContext, typ string) error {
	port := nctx.Args["port"]
	remote := nctx.Args["remote"]
	if remote == "" {
		if isRange(port) {
			return fmt.Errorf("port range needs an explicit --remote range")
		}
		remote = fmt.Sprint(30000 + randInt(30000))
	}
	p := frpc.Proxy{
		Name: argOr(nctx.Args["name"], fmt.Sprintf("laz-%s-%s", typ, port)),
		Type: typ, LocalIP: nctx.Args["host"], LocalPort: port,
		RemotePort: remote,
	}
	applyCommon(&p, nctx)
	return run(ctx, nctx, frpc.ClientConfig{Proxies: []frpc.Proxy{p}}, nil)
}

func runShare(ctx context.Context, nctx engine.NativeContext) error {
	typ := "s" + nctx.Args["type"] // tcp->stcp, udp->sudp
	if typ != "stcp" && typ != "sudp" {
		return fmt.Errorf("--type must be tcp or udp")
	}
	key := argOr(nctx.Args["key"], randomID(16))
	name := argOr(nctx.Args["name"], "laz-"+randomID(6))
	p := frpc.Proxy{
		Name: name, Type: typ,
		LocalIP: "127.0.0.1", LocalPort: nctx.Args["port"],
		SecretKey: key,
	}
	applyCommon(&p, nctx)
	return run(ctx, nctx, frpc.ClientConfig{Proxies: []frpc.Proxy{p}}, nil)
}

func runVisit(ctx context.Context, nctx engine.NativeContext) error {
	bindHost, bindPort := "127.0.0.1", 0
	if b := nctx.Args["bind"]; b != "" {
		h, ps, err := splitHostPort(b)
		if err != nil {
			return err
		}
		bindHost = h
		bindPort, _ = strconv.Atoi(ps)
	}
	if bindPort == 0 {
		bindPort = 10000 + randInt(50000)
	}
	v := frpc.Visitor{
		Name: "laz-visit-" + nctx.Args["name"], Type: nctx.Args["type"],
		ServerName: nctx.Args["name"], SecretKey: nctx.Args["key"],
		BindAddr: bindHost, BindPort: bindPort,
	}
	return run(ctx, nctx, frpc.ClientConfig{Visitors: []frpc.Visitor{v}}, nil)
}

func runXTCP(ctx context.Context, nctx engine.NativeContext) error {
	key := argOr(nctx.Args["key"], randomID(16))
	name := argOr(nctx.Args["name"], "laz-"+randomID(6))
	p := frpc.Proxy{
		Name: name, Type: "xtcp",
		LocalIP: "127.0.0.1", LocalPort: nctx.Args["port"],
		SecretKey: key,
	}
	return run(ctx, nctx, frpc.ClientConfig{Proxies: []frpc.Proxy{p}}, nil)
}

func runDir(ctx context.Context, nctx engine.NativeContext) error {
	path, err := filepath.Abs(nctx.Args["path"])
	if err != nil {
		return err
	}
	sub := argOr(nctx.Args["subdomain"], "laz-"+randomID(6))
	p := frpc.Proxy{
		Name: sub, Type: "http", Subdomain: sub,
		Plugin: &frpc.Plugin{
			Type:         "static_file",
			LocalPath:    path,
			HTTPUser:     nctx.Args["user"],
			HTTPPassword: nctx.Args["pass"],
		},
	}
	return run(ctx, nctx, frpc.ClientConfig{Proxies: []frpc.Proxy{p}},
		func() string {
			return fmt.Sprintf("https://%s.loc.rest", p.Subdomain)
		})
}

func runSocks(ctx context.Context, nctx engine.NativeContext) error {
	remote := nctx.Args["remote"]
	if remote == "" {
		remote = fmt.Sprint(30000 + randInt(30000))
	}
	p := frpc.Proxy{
		Name: "laz-socks", Type: "tcp", RemotePort: remote,
		Plugin: &frpc.Plugin{
			Type:     "socks5",
			Username: nctx.Args["user"],
			Password: nctx.Args["pass"],
		},
	}
	return run(ctx, nctx, frpc.ClientConfig{Proxies: []frpc.Proxy{p}}, nil)
}

// run renders config and launches frpc under signal+TTL context.
// target, when non-nil, prints the public endpoint line.
func run(ctx context.Context, nctx engine.NativeContext,
	cc frpc.ClientConfig, target func() string) error {

	cfg, err := config.Load(nctx.Args["server"], nctx.Args["token"],
		nctx.Args["transport"])
	if err != nil {
		return err
	}
	cc.Server, cc.Token, cc.Transport = cfg.Server, cfg.Token, cfg.Transport
	doc, err := frpc.Render(cc, cfg.DefaultServerPort)
	if err != nil {
		return err
	}

	if nctx.Args["render"] == "true" {
		_, err := nctx.Stdout.Write(doc)
		return err
	}

	bin, err := frpc.Ensure("")
	if err != nil {
		return err
	}

	var ttl time.Duration
	if s := nctx.Args["ttl"]; s != "" {
		if ttl, err = time.ParseDuration(s); err != nil {
			return fmt.Errorf("invalid --ttl: %w", err)
		}
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var tgt string
	if target != nil {
		tgt = target()
	}
	printBanner(nctx.Stdout, cc, tgt, nctx.Args["ttl"])
	return frpc.Run(ctx, bin, doc, ttl)
}

func applyCommon(p *frpc.Proxy, nctx engine.NativeContext) {
	p.UseEncryption = nctx.Args["encrypt"] == "true"
	p.UseCompression = nctx.Args["compress"] == "true"
	p.BandwidthLimit = nctx.Args["bandwidth"]
}

func runFRPCFetch(ctx context.Context, nctx engine.NativeContext) error {
	path, err := frpc.Ensure(nctx.Args["version"])
	if err != nil {
		return err
	}
	fmt.Fprintf(nctx.Stdout, "laz: frpc at %s\n", path)
	return nil
}

func showVersion(ctx context.Context, nctx engine.NativeContext) error {
	fmt.Fprintf(nctx.Stdout, "laz %s (%s)\n", version, commit)
	return nil
}

func argOr(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

func isRange(s string) bool {
	for _, c := range s {
		if c == '-' {
			return true
		}
	}
	return false
}

func splitHostPort(s string) (string, string, error) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return s[:i], s[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("bind must be host:port, got %q", s)
}

func randomID(n int) string {
	const abc = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(abc))))
		b[i] = abc[v.Int64()]
	}
	return string(b)
}

func randInt(max int) int {
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(v.Int64())
}
