package transfer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PartialState struct {
	Code          string `json:"code"`
	FileName      string `json:"file_name"`
	LastSeq       uint32 `json:"last_seq"`
	BytesReceived int64  `json:"bytes_received"`
	TotalSize     int64  `json:"total_size"`
}

func partialPath(outputDir, fileName string) string {
	return filepath.Join(outputDir, fileName+".tossit-partial")
}

func SavePartialState(outputDir string, state PartialState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(partialPath(outputDir, state.FileName), data, 0o600)
}

func LoadPartialState(outputDir, fileName string) (*PartialState, error) {
	data, err := os.ReadFile(partialPath(outputDir, fileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state PartialState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func ClearPartialState(outputDir, fileName string) {
	_ = os.Remove(partialPath(outputDir, fileName))
}
