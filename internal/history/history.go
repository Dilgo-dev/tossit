package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Direction string

const (
	Sent     Direction = "sent"
	Received Direction = "received"
)

type Entry struct {
	Direction Direction `json:"direction"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Code      string    `json:"code"`
	Time      time.Time `json:"time"`
}

func dataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "tossit")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "tossit")
}

func historyPath() string {
	return filepath.Join(dataDir(), "history.jsonl")
}

func Add(e Entry) {
	dir := dataDir()
	_ = os.MkdirAll(dir, 0o750)

	f, err := os.OpenFile(historyPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_, _ = f.Write(data)
}

func Load() ([]Entry, error) {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func Clear() error {
	return os.Remove(historyPath())
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
