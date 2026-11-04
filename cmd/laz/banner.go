package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/quonaro/laz/internal/frpc"
)

const (
	cReset = "\x1b[0m"
	cBold  = "\x1b[1m"
	cDim   = "\x1b[2m"
	cCyan  = "\x1b[36m"
	cGreen = "\x1b[32m"
)

// printBanner draws a boxed summary of the tunnel before frpc starts.
// target is the public endpoint for http-like proxies; when empty it is
// derived from the server host + remote port.
func printBanner(w io.Writer, cc frpc.ClientConfig, target, ttl string) {
	tag := "tunnel"
	var rows [][2]string
	var opts []string

	if ttl != "" {
		opts = append(opts, "ttl "+ttl)
	}
	host := cc.Server
	if h, _, err := net.SplitHostPort(cc.Server); err == nil {
		host = h
	}
	foot := "server  " + cc.Server
	if cc.Transport != "" {
		foot += " (" + cc.Transport + ")"
	}

	switch {
	case len(cc.Proxies) > 0:
		p := cc.Proxies[0]
		tag = p.Type
		switch {
		case p.Type == "stcp" || p.Type == "sudp" || p.Type == "xtcp":
			rows = [][2]string{
				{"local", p.LocalIP + ":" + p.LocalPort},
				{"name", p.Name},
				{"key", p.SecretKey},
				{"peer", fmt.Sprintf("laz visit %s -k %s -y %s", p.Name, p.SecretKey, p.Type)},
			}
		case p.Plugin != nil && p.Plugin.Type == "static_file":
			rows = [][2]string{
				{"dir", p.Plugin.LocalPath},
				{"↓", ""},
				{"public", target},
			}
		case p.Plugin != nil:
			rows = [][2]string{
				{"plugin", p.Plugin.Type},
				{"↓", ""},
				{"public", host + ":" + p.RemotePort},
			}
		default:
			if target == "" {
				target = host + ":" + p.RemotePort
			}
			rows = [][2]string{
				{"local", p.LocalIP + ":" + p.LocalPort},
				{"↓", ""},
				{"public", target},
			}
		}
		if p.UseEncryption {
			opts = append(opts, "encrypt")
		}
		if p.UseCompression {
			opts = append(opts, "compress")
		}
		if p.BandwidthLimit != "" {
			opts = append(opts, "bw="+p.BandwidthLimit)
		}
	case len(cc.Visitors) > 0:
		v := cc.Visitors[0]
		tag = v.Type
		rows = [][2]string{
			{"visit", v.ServerName},
			{"↓", ""},
			{"bind", fmt.Sprintf("%s:%d", v.BindAddr, v.BindPort)},
		}
	}
	drawBanner(w, tag, rows, foot, strings.Join(opts, "  "))
}

// drawBanner renders a double-bordered box: a title/tag header,
// label/value body rows and a server/options footer.
func drawBanner(w io.Writer, tag string, rows [][2]string, foot, opts string) {
	tty := isTTY(w)

	lw := 0
	for _, r := range rows {
		if r[1] != "" && len(r[0]) > lw {
			lw = len(r[0])
		}
	}
	footRow := foot
	if opts != "" {
		footRow += "   " + opts
	}

	const title = "laz tunnel"
	inner := runeLen(title) + 4 + runeLen(tag)
	for _, r := range rows {
		n := runeLen(r[0])
		if r[1] != "" {
			n = lw + 2 + runeLen(r[1])
		}
		inner = max(inner, n)
	}
	inner = min(max(inner, runeLen(footRow), 44), 96)

	bar := strings.Repeat("═", inner+6)
	border := func(l, r string) {
		if tty {
			fmt.Fprintf(w, "%s%s%s%s%s\n", cCyan, l, bar, r, cReset)
		} else {
			fmt.Fprintf(w, "%s%s%s\n", l, bar, r)
		}
	}
	row := func(s string, vis int) {
		fmt.Fprintf(w, "║   %s%s   ║\n", s, strings.Repeat(" ", inner-vis))
	}

	border("╔", "╗")
	gap := inner - runeLen(title) - runeLen(tag)
	if tty {
		fmt.Fprintf(w, "║   %s%s%s%s%s%s%s   ║\n",
			cBold, title, cReset, strings.Repeat(" ", gap), cCyan, tag, cReset)
	} else {
		row(title+strings.Repeat(" ", gap)+tag, inner)
	}
	border("╠", "╣")
	for _, r := range rows {
		if r[1] == "" {
			s := truncate(r[0], inner)
			vis := runeLen(s)
			if tty {
				s = cDim + s + cReset
			}
			row(s, vis)
			continue
		}
		lab := fmt.Sprintf("%-*s  ", lw, r[0])
		val := truncate(r[1], inner-len(lab))
		out := lab + val
		if tty {
			out = cDim + lab + cReset + val
			switch r[0] {
			case "public", "bind":
				out = cDim + lab + cReset + cBold + cGreen + val + cReset
			case "key", "peer":
				out = cDim + lab + cReset + cGreen + val + cReset
			}
		}
		row(out, len(lab)+runeLen(val))
	}
	border("╠", "╣")
	fr := truncate(footRow, inner)
	vis := runeLen(fr)
	if tty {
		fr = cDim + fr + cReset
	}
	row(fr, vis)
	border("╚", "╝")
	fmt.Fprintln(w)
}

func runeLen(s string) int { return len([]rune(s)) }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
