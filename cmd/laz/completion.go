package main

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/quonaro/lota/engine"
)

// handleCompletion intercepts the hidden `__complete` invocation used by
// generated shell scripts: `laz __complete <line> <point>`.
// Returns true when the request was a completion and has been served.
func handleCompletion(app *engine.App, out io.Writer, args []string) bool {
	if len(args) < 3 || args[0] != "__complete" {
		return false
	}
	point, err := strconv.Atoi(args[2])
	if err != nil {
		return true
	}
	options, hint, err := app.Complete(args[1], point, "laz")
	if err != nil {
		return true
	}
	for _, o := range options {
		fmt.Fprintln(out, o)
	}
	if hint != "" {
		fmt.Fprintln(out, "__hint__:"+hint)
	}
	return true
}

func runCompletion(ctx context.Context, nctx engine.NativeContext) error {
	script, err := engine.GetCompletionScript(nctx.Args["shell"], "laz")
	if err != nil {
		return err
	}
	fmt.Fprint(nctx.Stdout, script)
	return nil
}
