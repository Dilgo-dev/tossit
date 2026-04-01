package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const repo = "Dilgo-dev/tossit"

type release struct {
	TagName string `json:"tag_name"`
}

func Check(current string) (string, bool, error) {
	resp, err := http.Get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

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
		fmt.Printf("Already on latest version (%s)\n", current)
		return nil
	}

	fmt.Printf("Updating %s -> %s\n", current, latest)

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/tossit-%s-%s%s",
		repo, latest, goos, goarch, ext)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = resolveSymlink(exe)
	if err != nil {
		return err
	}

	tmp := exe + ".new"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()

	if err := os.Rename(tmp, exe); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("Updated to %s\n", latest)
	return nil
}

func resolveSymlink(path string) (string, error) {
	resolved, err := os.Readlink(path)
	if err != nil {
		return path, nil
	}
	if !strings.HasPrefix(resolved, "/") {
		dir := path[:strings.LastIndex(path, "/")+1]
		resolved = dir + resolved
	}
	return resolved, nil
}
