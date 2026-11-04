#!/bin/sh
set -e

OWNER="${OWNER:-quonaro}"
REPO="${REPO:-Laz}"
VERSION="${VERSION:-latest}"
BIN_NAME="${BIN_NAME:-laz}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
FRP_VERSION="${FRP_VERSION:-0.71.1}"
WITH_FRPC="${WITH_FRPC:-1}"
WITH_COMPLETION="${WITH_COMPLETION:-1}"

if [ -t 1 ]; then
	GREEN='\033[0;32m' YELLOW='\033[1;33m' RED='\033[0;31m' NC='\033[0m'
else
	GREEN='' YELLOW='' RED='' NC=''
fi

info() { printf "${GREEN}==>${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}WARN:${NC} %s\n" "$*" >&2; }
die()  { printf "${RED}ERROR:${NC} %s\n" "$*" >&2; exit 1; }

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
	x86_64|amd64)  ARCH=amd64 ;;
	aarch64|arm64) ARCH=arm64 ;;
	*) die "unsupported architecture: $ARCH" ;;
esac
case "$OS" in linux|darwin) ;; *) die "unsupported OS: $OS" ;; esac

fetch() { # url file
	if command -v curl >/dev/null 2>&1; then curl -fsSL -o "$2" "$1";
	elif command -v wget >/dev/null 2>&1; then wget -q -O "$2" "$1";
	else die "curl or wget is required"; fi
}

if [ "$VERSION" = "latest" ]; then
	VERSION=$(fetch "https://api.github.com/repos/$OWNER/$REPO/releases/latest" - \
		| sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1) \
		|| die "could not resolve latest release"
	[ -n "$VERSION" ] || die "no releases found for $OWNER/$REPO"
fi

mkdir -p "$INSTALL_DIR"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

info "downloading laz $VERSION ($OS/$ARCH)"
fetch "https://github.com/$OWNER/$REPO/releases/download/$VERSION/laz-$OS-$ARCH" "$TMP/laz" \
	|| die "release asset laz-$OS-$ARCH not found"
install -m 755 "$TMP/laz" "$INSTALL_DIR/laz"

if [ "$WITH_FRPC" = "1" ] && ! command -v frpc >/dev/null 2>&1; then
	info "downloading frpc $FRP_VERSION"
	fetch "https://github.com/fatedier/frp/releases/download/v$FRP_VERSION/frp_${FRP_VERSION}_${OS}_${ARCH}.tar.gz" "$TMP/frp.tar.gz" \
		|| die "could not download frp release"
	tar xzf "$TMP/frp.tar.gz" -C "$TMP"
	install -m 755 "$TMP/frp_${FRP_VERSION}_${OS}_${ARCH}/frpc" "$INSTALL_DIR/frpc"
fi

install_completion() { # <shell>
	case "$1" in
	fish)
		dir="$HOME/.config/fish/completions"
		mkdir -p "$dir" \
			&& "$INSTALL_DIR/laz" completion fish > "$dir/laz.fish" \
			&& info "completion: $dir/laz.fish"
		;;
	bash|zsh)
		rc="$HOME/.$1rc"
		line='command -v laz >/dev/null 2>&1 && eval "$(laz completion '"$1"')"'
		[ "$1" = zsh ] && line='command -v laz >/dev/null 2>&1 && { autoload -Uz compinit && compinit; eval "$(laz completion zsh)"; }'
		if ! grep -qF 'laz completion' "$rc" 2>/dev/null; then
			printf '\n# laz shell completion\n%s\n' "$line" >> "$rc" \
				&& info "completion: appended to $rc (restart shell)"
		fi
		;;
	esac
}

if [ "$WITH_COMPLETION" = "1" ]; then
	install_completion "$(basename "${SHELL:-}")" || warn "completion install skipped"
fi

case ":$PATH:" in *":$INSTALL_DIR:"*) ;; *)
	warn "$INSTALL_DIR is not in PATH" ;;
esac

info "installed: $INSTALL_DIR/laz$([ -f "$INSTALL_DIR/frpc" ] && echo " + frpc")"
echo "  next: laz http 3000 --server tunnel.example.com --token <token>"
