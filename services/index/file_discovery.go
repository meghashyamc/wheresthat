package index

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileInfo struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
	IsText  bool
}

func discoverFiles(rootPath string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		fileInfo := FileInfo{
			Path:    path,
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		fileInfo.IsText = isTextFile(path)

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

func isTextFile(path string) bool {
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".js": true,
		".py": true, ".java": true, ".cpp": true, ".c": true,
		".html": true, ".css": true, ".json": true, ".xml": true,
		".yaml": true, ".yml": true, ".ini": true, ".conf": true,
		".doc": true, ".xlsx": true, ".docx": true, ".pptx": true,
		".csv": true, ".tsv": true, ".sql": true, ".pdf": true, ".cs": true,
	}

	ext := strings.ToLower(filepath.Ext(path))
	return textExtensions[ext]
}
