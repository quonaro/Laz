package frpc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Run renders the config, writes it to a temp file and executes frpc
// until ctx is cancelled or the TTL elapses.
func Run(ctx context.Context, binary string, cfg []byte, ttl time.Duration) error {
	dir, err := os.MkdirTemp("", "laz-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "frpc.toml")
	if err := os.WriteFile(path, cfg, 0o600); err != nil {
		return err
	}

	if ttl > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ttl)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, binary, "-c", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("frpc: %w", err)
	}
	return nil
}
