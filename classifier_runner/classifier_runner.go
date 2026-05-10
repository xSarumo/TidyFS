package classifier_runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func RunClassifier(projectRoot string) error {

	pythonPath := filepath.Join(projectRoot, ".venv", "bin", "python")
	scriptPath := filepath.Join(projectRoot, "classifier", "classifier.py")

	if _, err := os.Stat(pythonPath); err != nil {
		return fmt.Errorf("venv python not found: %s\nrun: make py-deps", pythonPath)
	}

	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("classifier script not found: %s", scriptPath)
	}

	cmd := exec.Command(pythonPath, scriptPath)
	cmd.Dir = projectRoot

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"classifier failed: %w\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout.String(),
			stderr.String(),
		)
	}

	return nil
}
