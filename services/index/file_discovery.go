package index

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileInfo struct {
	Path        string
	Name        string
	Size        int64
	ModTime     time.Time
	IsText      bool
	ContentType string
}

// Step 1: Discover all files recursively
func discoverFiles(rootPath string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		fileInfo := FileInfo{
			Path:    path,
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		// Determine if file is text or binary
		fileInfo.IsText = isTextFile(path)
		fileInfo.ContentType = getContentType(path)

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

func isTextFile(path string) bool {
	// Simple heuristic - check extension
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".js": true,
		".py": true, ".java": true, ".cpp": true, ".c": true,
		".html": true, ".css": true, ".json": true, ".xml": true,
		".yaml": true, ".yml": true, ".ini": true, ".conf": true,
	}

	ext := strings.ToLower(filepath.Ext(path))
	return textExtensions[ext]
}

func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md":
		return "text"
	case ".go", ".js", ".py", ".java":
		return "code"
	case ".jpg", ".png", ".gif":
		return "image"
	case ".mp4", ".avi", ".mov":
		return "video"
	case ".pdf":
		return "document"
	default:
		return "unknown"
	}
}
