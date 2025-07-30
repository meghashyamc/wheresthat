package index

import (
	"io"
	"os"

	"github.com/meghashyamc/wheresthat/db/searchdb"
)

const maxContentExtractionSize = 50 * 1024 * 1024 // 50MB limit

func extractContent(fileInfo FileInfo) (*searchdb.Document, error) {
	doc := &searchdb.Document{
		ID:      fileInfo.Path,
		Path:    fileInfo.Path,
		Name:    fileInfo.Name,
		Size:    fileInfo.Size,
		ModTime: fileInfo.ModTime,
	}

	if fileInfo.IsText {

		file, err := os.Open(fileInfo.Path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		content, err := readTextContent(file, fileInfo.Size)
		if err != nil {
			return nil, err
		}
		doc.Content = content
	}

	return doc, nil
}

func readTextContent(reader io.Reader, fileSize int64) (string, error) {
	limitedReader := reader

	if fileSize > maxContentExtractionSize {
		// Limit reader to maxContentExtractionSize bytes
		limitedReader = io.LimitReader(reader, maxContentExtractionSize)
	}

	// Read entire content for smaller files
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
