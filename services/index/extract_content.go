package index

import (
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/meghashyamc/wheresthat/db/searchdb"
)

const maxContentExtractionSize = 50 * 1024 * 1024 // 50MB limit

func extractContent(fileInfo FileInfo) (*searchdb.Document, error) {
	doc := &searchdb.Document{
		ID:      uuid.New().String(),
		Path:    fileInfo.Path,
		Name:    fileInfo.Name,
		Size:    fileInfo.Size,
		ModTime: fileInfo.ModTime,
	}

	if fileInfo.IsText {
		content, err := readTextFile(fileInfo.Path)
		if err != nil {
			return nil, err
		}
		doc.Content = content
	}

	return doc, nil
}

func readTextFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	if stat.Size() > maxContentExtractionSize {
		// For large files, read only first portion
		buffer := make([]byte, maxContentExtractionSize)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		return string(buffer[:n]), nil
	}

	// Read entire file for smaller files
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
