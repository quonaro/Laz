package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/quonaro/laz/internal/config"
	"github.com/quonaro/laz/internal/frpc"
	"github.com/quonaro/lota/engine"
)

func runHTTP(ctx context.Context, nctx engine.NativeContext) error {
	port, _ := strconv.Atoi(nctx.Args["port"])
	sub := nctx.Args["subdomain"]
	if sub == "" {
		sub = "laz-" + randomID(6)
	}
	return run(ctx, nctx, frpc.Proxy{
		Name: sub, Type: "http",
		LocalIP: nctx.Args["host"], LocalPort: port,
		Subdomain: sub,
	})
}

func runTCP(ctx context.Context, nctx engine.NativeContext) error {
	return run(ctx, nctx, rawProxy("tcp", nctx))
}

func runUDP(ctx context.Context, nctx engine.NativeContext) error {
	return run(ctx, nctx, rawProxy("udp", nctx))
}

func rawProxy(typ string, nctx engine.NativeContext) frpc.Proxy {
	port, _ := strconv.Atoi(nctx.Args["port"])
	remote, _ := strconv.Atoi(nctx.Args["remote"])
	if remote == 0 {
		remote = 30000 + randInt(30000)
	}
	return frpc.Proxy{
		Name: fmt.Sprintf("laz-%s-%d", typ, port), Type: typ,
		LocalIP: nctx.Args["host"], LocalPort: port, RemotePort: remote,
	}
}

func run(ctx context.Context, nctx engine.NativeContext, p frpc.Proxy) error {
	cfg, err := config.Load(nctx.Args["server"], nctx.Args["token"])
	if err != nil {
		return err
	}
	bin, err := frpc.Ensure("")
	if err != nil {
		return err
	}
	doc, err := frpc.Render(cfg.Server, cfg.Token, cfg.Transport,
		cfg.DefaultServerPort, []frpc.Proxy{p})
	if err != nil {
		return err
	}

	ttl, err := time.ParseDuration(nctx.Args["ttl"])
	if nctx.Args["ttl"] == "" {
		ttl, err = 0, nil
	}
	if err != nil {
		return fmt.Errorf("invalid --ttl: %w", err)
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(nctx.Stdout, "laz: %s tunnel -> %s (local %s:%d)\n",
		p.Type, describeTarget(cfg, p), p.LocalIP, p.LocalPort)
	if ttl > 0 {
		fmt.Fprintf(nctx.Stdout, "laz: ttl %s\n", ttl)
	}
	return frpc.Run(ctx, bin, doc, ttl)
}

func describeTarget(cfg *config.Config, p frpc.Proxy) string {
	if p.Type == "http" {
		return fmt.Sprintf("%s.<domain of %s>", p.Subdomain, cfg.Server)
	}
	return fmt.Sprintf("%s:%d", cfg.Server, p.RemotePort)
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
