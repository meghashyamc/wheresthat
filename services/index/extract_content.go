package index

import (
	"io"
	"os"
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID      string // Unique identifier (could be file path hash)
	Path    string // Full file path
	Name    string // Just filename
	Content string // Extracted text content (for text files)
	Size    int64  // File size
	ModTime time.Time
	Type    string  // File type category
	Tokens  []Token // Processed tokens with positions
}

type Token struct {
	Text     string
	Position int // Character position in original content
	Line     int // Line number
	Column   int // Column number
}

func extractContent(fileInfo FileInfo) (*Document, error) {
	doc := &Document{
		ID:      uuid.New().String(),
		Path:    fileInfo.Path,
		Name:    fileInfo.Name,
		Size:    fileInfo.Size,
		ModTime: fileInfo.ModTime,
		Type:    fileInfo.ContentType,
	}

	if fileInfo.IsText {
		content, err := readTextFile(fileInfo.Path)
		if err != nil {
			return nil, err
		}
		doc.Content = content
		doc.Tokens = tokenize(content)
	} else {
		// For binary files, only tokenize the filename
		doc.Tokens = tokenize(fileInfo.Name)
	}

	return doc, nil
}

func readTextFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read file with size limit to prevent memory issues
	const maxFileSize = 10 * 1024 * 1024 // 10MB limit

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	if stat.Size() > maxFileSize {
		// For large files, read only first portion
		buffer := make([]byte, maxFileSize)
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
