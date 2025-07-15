package index

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meghashyamc/wheresthat/db/kvdb"
)

type FileInfo struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
	IsText  bool
}

func (s *Service) discoverModifiedFiles(rootPath string, lastIndexTime time.Time) ([]FileInfo, error) {
	var modifiedFiles []FileInfo

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		fileModTime := info.ModTime()

		if s.shouldFileBeIndexed(path, fileModTime, lastIndexTime) {
			fileInfo := FileInfo{
				Path:    path,
				Name:    info.Name(),
				Size:    info.Size(),
				ModTime: fileModTime,
			}

			fileInfo.IsText = isTextFile(path)
			modifiedFiles = append(modifiedFiles, fileInfo)
		}

		return nil
	})

	return modifiedFiles, err
}

func (s *Service) shouldFileBeIndexed(path string, fileModTime time.Time, lastIndexTime time.Time) bool {

	// Check if this file was indexed before
	metadata, err := s.getFileMetadata(path)
	if err != nil {
		var notFoundErr *kvdb.NotFoundError
		var invalidKeyErr *kvdb.InvalidKeyError

		switch {
		// File not found in database, should be indexed
		case errors.As(err, &notFoundErr):
			return true
			// Invalid key, log error and index
		case errors.As(err, &invalidKeyErr):
			s.logger.Error("invalid key for file path", "key", path, "err", err.Error())
			return true
		// Unknown error, log error and index
		default:
			s.logger.Error("failed to get metadata", "path", path, "err", err.Error())
			return true
		}
	}

	// File was indexed before, check if it was modified since
	if fileModTime.After(metadata.LastIndexed) {
		return true
	}

	// File is already indexed and has not been modified after indexing
	return false
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
