package exporter

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func SaveJSON(path string, data any) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, bytes, 0644)
}
