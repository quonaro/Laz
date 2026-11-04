package frpc

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// PinnedVersion is the frp release laz downloads when no frpc is found.
const PinnedVersion = "0.71.1"

// Locate finds an frpc binary: $LAZ_FRPC > PATH > laz data dir.
// Returns an empty string when nothing is found.
func Locate() string {
	if p := os.Getenv("LAZ_FRPC"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("frpc"); err == nil {
		return p
	}
	if p := managedPath(); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// managedPath is where laz keeps its own downloaded frpc.
func managedPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "laz", "frpc")
}

// Ensure returns a usable frpc path, downloading the pinned release into
// the laz cache dir when no binary is found.
func Ensure(version string) (string, error) {
	if version == "" {
		if p := Locate(); p != "" {
			return p, nil
		}
		version = PinnedVersion
	}
	dest := managedPath()
	if dest == "" {
		return "", fmt.Errorf("cannot determine cache dir; install frpc or set LAZ_FRPC")
	}
	if err := download(dest, version); err != nil {
		return "", err
	}
	return dest, nil
}

// download fetches frpc from the frp GitHub release tarball.
func download(dest, version string) error {
	asset := fmt.Sprintf("frp_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/fatedier/frp/releases/download/v%s/%s", version, asset)

	resp, err := http.Get(url) //nolint:gosec // fixed GitHub release URL
	if err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %s", asset, resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if filepath.Base(hdr.Name) != "frpc" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		tmp := dest + ".tmp"
		f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return err
		}
		f.Close()
		return os.Rename(tmp, dest)
	}
	return fmt.Errorf("frpc not found in %s", asset)
}
