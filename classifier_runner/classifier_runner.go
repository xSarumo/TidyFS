package classifier_runner

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const ollamaURL = "http://127.0.0.1:11434"

func RunClassifier(projectRoot string, mode string) error {
	var ollamaCmd *exec.Cmd

	if mode == "llm" {
		running := isOllamaRunning()

		if !running {
			var err error
			ollamaCmd, err = startOllama()
			if err != nil {
				return err
			}

			defer stopOllama(ollamaCmd)

			if err := waitForOllama(20 * time.Second); err != nil {
				return err
			}
		}
	}

	var modeFile strings.Builder
	modeFile.WriteString("classifier_")
	modeFile.WriteString(mode)
	modeFile.WriteString(".py")

	pythonPath := filepath.Join(projectRoot, ".venv", "bin", "python")
	scriptPath := filepath.Join(projectRoot, "classifier", modeFile.String())

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

func isOllamaRunning() bool {
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	resp, err := client.Get(ollamaURL + "/api/tags")
	if err != nil {
		return false
	}

	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func startOllama() (*exec.Cmd, error) {
	if _, err := exec.LookPath("ollama"); err != nil {
		return nil, fmt.Errorf("ollama not found in PATH; install it or use TF_IDF mode")
	}

	cmd := exec.Command("ollama", "serve")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ollama serve: %w\nstderr:\n%s", err, stderr.String())
	}

	return cmd, nil
}

func waitForOllama(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if isOllamaRunning() {
			return nil
		}

		time.Sleep(300 * time.Millisecond)
	}

	return fmt.Errorf("ollama did not become ready within %s", timeout)
}

func stopOllama(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}
