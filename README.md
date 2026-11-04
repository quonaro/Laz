# ⚡ Laz

Tiny CLI that generates an ephemeral [frp](https://github.com/fatedier/frp) client
config and connects. One command, one tunnel — no hand-edited `frpc.toml`.

```sh
laz http 3000 --subdomain myapp        # https://myapp.your-frps-domain
laz tcp 22 --remote 2222               # ssh via tunnel.example.com:2222
laz udp 19132                          # game server
laz http 3000 --ttl 1h                 # dies on its own after an hour
laz http 3000 -s myapp --render > frpc.toml   # config only, no tunnel
```

`--render` prints the generated `frpc.toml` to stdout instead of running
`frpc` — handy for long-lived tunnels managed by systemd or Docker
(`docker run -v ./frpc.toml:/etc/frp/frpc.toml fatedier/frpc`).

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/quonaro/Laz/main/public/install.sh | sh
```

Installs `laz` (and `frpc`, if missing) into `~/.local/bin`, plus shell
completion for bash/zsh/fish (`WITH_COMPLETION=0` to skip). Manual:
`laz completion <shell>`.

## Configure

Resolution order: flag > env > config file (`./laz.yaml` or
`~/.config/laz/config.yaml`).

```yaml
# ~/.config/laz/config.yaml
server: tunnel.example.com:7000
token: <frps auth token>
transport: websocket   # optional: tcp, quic, kcp, websocket
```

Or ad hoc: `laz http 3000 --server tunnel.example.com --token <token>`.

## Server side

Needs a reachable [frps](https://github.com/fatedier/frp) with `auth.method = "token"`.
`laz` only ever talks to the frp control port; the server config stays static —
all tunnel declarations live client-side and die with the process.

## Build

```sh
lota build.local   # ./laz for this platform
lota build.all     # cross-compiled release assets into dist/
```
