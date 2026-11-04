package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/quonaro/lota/engine"
)

const releaseRepo = "quonaro/Laz"

// latestRelease returns the tag of the newest GitHub release.
func latestRelease() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", releaseRepo)
	resp, err := http.Get(url) //nolint:gosec // fixed repo URL
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api: %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no releases found")
	}
	return rel.TagName, nil
}

// selfReplace downloads the release asset for this platform and atomically
// swaps it over the running binary.
func selfReplace(tag string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}

	asset := fmt.Sprintf("laz-%s-%s", runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", releaseRepo, tag, asset)
	resp, err := http.Get(url) //nolint:gosec // fixed repo URL
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("asset %s: %s", asset, resp.Status)
	}

	tmp, err := os.CreateTemp(filepath.Dir(exe), ".laz-update-*")
	if err != nil {
		return fmt.Errorf("cannot write next to %s: %w", exe, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	return os.Rename(tmp.Name(), exe)
}

func runUpdate(ctx context.Context, nctx engine.NativeContext) error {
	tag, err := latestRelease()
	if err != nil {
		return err
	}
	if tag == version {
		fmt.Fprintf(nctx.Stdout, "laz is up to date (%s)\n", version)
		return nil
	}
	if nctx.Args["check"] == "true" {
		fmt.Fprintf(nctx.Stdout, "update available: %s -> %s\n", version, tag)
		return nil
	}
	if err := selfReplace(tag); err != nil {
		return err
	}
	fmt.Fprintf(nctx.Stdout, "laz updated: %s -> %s\n", version, tag)
	return nil
}
