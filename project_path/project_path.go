package projectpath

import (
	"path/filepath"
	"runtime"
)

func Root() string {
	_, filename, _, _ := runtime.Caller(0)

	return filepath.Dir(filepath.Dir(filename))
}

func FilesJSON() string {
	return filepath.Join(Root(), "files", "files.json")
}

func ClassifiedFilesJSON() string {
	return filepath.Join(Root(), "files", "classified_files.json")
}
