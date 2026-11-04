# Laz

Thin CLI wrapper over `frpc`: renders an ephemeral `frpc.toml` from CLI args
and runs the tunnel until Ctrl+C or `--ttl` expiry. Server stays static —
all dynamics live client-side.

## Commands

Tasks are in `lota.yml` (`lota <task>`): `build.local`, `build.all`, `dev`, `push`.

```sh
go build ./... && go vet ./... && go test ./...
```

## Architecture

- `cmd/laz/main.go` — embedded Lota engine bootstrap (`cli.yml` + native
  handlers), same pattern as acp2api.
- `cmd/laz/cli.yml` — command declarations (`http`, `tcp`, `udp`, `frpc`,
  `version`); `native: true` handlers.
- `cmd/laz/tunnel.go` — native handlers: args → `frpc.Proxy` → render → run.
- `internal/config` — server/token resolution: flag > env (`LAZ_*`) >
  `~/.config/laz/config.yaml` > `./laz.yaml`.
- `internal/frpc` — `config.go` renders frpc.toml, `binary.go` locates or
  downloads a pinned frpc (`$LAZ_FRPC` > PATH > cache), `run.go` spawns frpc
  under a TTL-aware context.
- `public/install.sh` — `curl | sh` installer served via raw.githubusercontent.

## Rules

- Non-test source files: max 300 lines.
- Stdlib only where practical; `gopkg.in/yaml.v3` for config.
- Commit messages: conventional commits, no bot trailers.
- English for code, comments, docs.
