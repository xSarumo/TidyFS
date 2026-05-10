package scanner

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	stdunicode "unicode"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type Scanner struct {
	supportExtentions map[string]bool
}

func NewScanner() *Scanner {
	return &Scanner{
		supportExtentions: map[string]bool{
			".pdf":  true,
			".txt":  true,
			".doc":  true,
			".docx": true,
			".md":   true},
	}
}

type File struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Ext  string `json:"ext"`
	Text string `json:"text"`
}

func (sc *Scanner) ScanDirs(root string) ([]File, error) {
	root, err := ExpandPath(root)
	if err != nil {
		return nil, err
	}

	var files []File

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))

		if sc.supportExtentions[ext] {
			files = append(files, File{
				Path: path,
				Ext:  ext,
				Name: info.Name(),
				Text: readTextPreview(path, ext),
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func readTextPreview(path, ext string) string {
	previewData := 4 * 1024

	switch ext {
	case ".md", ".txt":
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}

		if len(data) > previewData {
			data = data[:previewData]
		}

		text := decodeText(data)
		return cleanText(text)
	default:
		return ""
	}
}

func decodeText(data []byte) string {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder()
		result, _, err := transform.Bytes(decoder, data)
		if err == nil {
			return string(result)
		}
	}

	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		decoder := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder()
		result, _, err := transform.Bytes(decoder, data)
		if err == nil {
			return string(result)
		}
	}

	if bytes.Count(data, []byte{0x00}) > len(data)/4 {
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		result, _, err := transform.Bytes(decoder, data)
		if err == nil {
			return string(result)
		}
	}

	return string(data)
}

func cleanText(s string) string {
	var b strings.Builder

	for _, r := range s {
		if r == '\n' || r == '\t' || r == '\r' {
			b.WriteRune(r)
			continue
		}

		if stdunicode.IsControl(r) {
			continue
		}

		b.WriteRune(r)
	}

	return strings.TrimSpace(b.String())
}

func ExpandPath(path string) (string, error) {
	path = strings.TrimSpace(path)

	if path == "" {
		return "", os.ErrInvalid
	}

	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		path = home
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		path = filepath.Join(home, path[2:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return "", fmt.Errorf("source path is not a directory: %s", absPath)
	}

	return absPath, nil
}
