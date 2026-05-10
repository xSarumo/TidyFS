package carrier

import (
	projectpath "TidyFS/project_path"
	"TidyFS/scanner"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func FileSystemCarrierRun(source, target, command string) error {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" || strings.TrimSpace(command) == "" {
		return fmt.Errorf("source, target or command is empty")
	}

	switch command {
	case "move":
		return carry(moveFile, source, target)

	case "copy":
		return carry(copyFile, source, target)

	default:
		return fmt.Errorf("unknown filesystem command: %s", command)
	}
}

func carry(operation func(string, string) error, source, target string) error {
	source, err := scanner.ExpandPath(source)
	if err != nil {
		return err
	}

	target, err = scanner.ExpandPath(target)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(projectpath.ClassifiedFilesJSON())
	if err != nil {
		return err
	}

	var classifiedFiles []*ClassifiedFile
	if err := json.Unmarshal(data, &classifiedFiles); err != nil {
		return err
	}

	for _, file := range classifiedFiles {
		if file == nil {
			return fmt.Errorf("file is nil")
		}

		if strings.TrimSpace(file.Path) == "" {
			return fmt.Errorf("file path is empty")
		}

		if strings.TrimSpace(file.Category) == "" {
			return fmt.Errorf("file category is empty: %s", file.Path)
		}

		if strings.TrimSpace(file.Name) == "" {
			return fmt.Errorf("file name is empty: %s", file.Path)
		}

		sourcePath := file.Path
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(source, sourcePath)
		}

		targetPath := filepath.Join(target, file.Category, file.Name)

		targetPath, err = uniqueTargetPath(targetPath)
		if err != nil {
			return err
		}

		if err := operation(sourcePath, targetPath); err != nil {
			return err
		}
	}

	return nil
}

func moveFile(sourcePath string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	err := os.Rename(sourcePath, targetPath)
	if err == nil {
		return nil
	}

	if !isCrossDeviceError(err) {
		return err
	}

	if err := copyFile(sourcePath, targetPath); err != nil {
		return err
	}

	return os.Remove(sourcePath)
}

func copyFile(sourcePath string, targetPath string) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("source is a directory, expected file: %s", sourcePath)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	dst, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode())
	if err != nil {
		return err
	}

	copied := false
	defer func() {
		_ = dst.Close()

		if !copied {
			_ = os.Remove(targetPath)
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	if err := dst.Sync(); err != nil {
		return err
	}

	if err := os.Chtimes(targetPath, info.ModTime(), info.ModTime()); err != nil {
		return err
	}

	copied = true
	return nil
}

func uniqueTargetPath(targetPath string) (string, error) {
	if _, err := os.Stat(targetPath); err != nil {
		if os.IsNotExist(err) {
			return targetPath, nil
		}

		return "", err
	}

	dir := filepath.Dir(targetPath)
	ext := filepath.Ext(targetPath)
	name := strings.TrimSuffix(filepath.Base(targetPath), ext)

	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, i, ext))

		if _, err := os.Stat(candidate); err != nil {
			if os.IsNotExist(err) {
				return candidate, nil
			}

			return "", err
		}
	}
}

func isCrossDeviceError(err error) bool {
	return errors.Is(err, os.ErrInvalid) ||
		strings.Contains(strings.ToLower(err.Error()), "cross-device")
}
