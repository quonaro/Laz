// Command laz is a thin wrapper around frpc: it renders an ephemeral
// frpc.toml from CLI arguments and runs the tunnel until it is stopped
// or its TTL expires.
//
// The command line is the embedded Lota engine (cli.yml); every command
// is a native Go handler.
package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/quonaro/lota/engine"
)

//go:embed cli.yml
var cliYAML []byte

func main() {
	builder := engine.NewBuilder("laz", cliYAML)
	builder.RegisterNative("http", runHTTP)
	builder.RegisterNative("tcp", runTCP)
	builder.RegisterNative("udp", runUDP)
	builder.RegisterNative("frpc", runFRPCFetch)
	builder.RegisterNative("version", showVersion)
	builder.RegisterNative("completion", runCompletion)
	builder.WithOptions(engine.Options{AutoHelp: true})

	app, err := builder.Build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "laz:", err)
		os.Exit(1)
	}

	if handleCompletion(app, os.Stdout, os.Args[1:]) {
		return
	}

	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		var groupErr *engine.GroupError
		if errors.As(err, &groupErr) {
			app.PrintGroupHelp(groupErr.Groups)
			return
		}
		fmt.Fprintln(os.Stderr, "laz:", err)
		os.Exit(1)
	}
}
