package relay

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FileConfig struct {
	Port          string   `json:"port"`
	Storage       string   `json:"storage"`
	Expire        string   `json:"expire"`
	MaxSize       string   `json:"max_size"`
	RateLimit     *int     `json:"rate_limit"`
	AuthToken     string   `json:"auth_token"`
	AllowIPs      []string `json:"allow_ips"`
	UIEnabled     *bool   `json:"ui"`
	UIPassword    *string `json:"ui_password"`
	AdminPassword string  `json:"admin_password"`
}

func LoadConfig(path string) (FileConfig, error) {
	data, err := os.ReadFile(path) //nolint:gosec // config path from trusted CLI flag
	if err != nil {
		return FileConfig{}, err
	}
	var fc FileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return FileConfig{}, fmt.Errorf("invalid config: %w", err)
	}
	return fc, nil
}

func ParseSize(s string) int64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	multiplier := int64(1)
	switch {
	case strings.HasSuffix(s, "GB"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 2 * 1024 * 1024 * 1024
	}
	return n * multiplier
}

func FormatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%dGB", bytes/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%dMB", bytes/(1024*1024))
	default:
		return fmt.Sprintf("%dKB", bytes/1024)
	}
}
