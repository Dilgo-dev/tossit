package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Dilgo-dev/tossit/internal/color"
)

const repo = "Dilgo-dev/tossit"

type release struct {
	TagName string `json:"tag_name"`
}

func Check(current string) (string, bool, error) {
	resp, err := http.Get("https://api.github.com/repos/" + repo + "/releases/latest") //nolint:gosec // URL is constant
	if err != nil {
		return "", false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return "", false, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", false, err
	}

	latest := r.TagName
	if latest == "" {
		return "", false, nil
	}

	if current == "dev" || current != latest {
		return latest, true, nil
	}
	return latest, false, nil
}

func Run(current string) error {
	latest, hasUpdate, err := Check(current)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if !hasUpdate {
		fmt.Printf("%s Already on latest version (%s)\n", color.Green("OK"), current)
		return nil
	}

	fmt.Printf("Updating %s -> %s\n", color.Dim(current), color.BoldCyan(latest))

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/tossit-%s-%s%s",
		repo, latest, goos, goarch, ext)

	resp, err := http.Get(url) //nolint:gosec // URL built from trusted constants
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe = resolveSymlink(exe)

	tmp := exe + ".new"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec // path derived from own executable
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	_ = f.Close()

	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("%s Updated to %s\n", color.Green("OK"), color.BoldCyan(latest))
	return nil
}

func resolveSymlink(path string) string {
	resolved, err := os.Readlink(path)
	if err != nil {
		return path
	}
	if !strings.HasPrefix(resolved, "/") {
		dir := path[:strings.LastIndex(path, "/")+1]
		resolved = dir + resolved
	}
	return resolved
}
