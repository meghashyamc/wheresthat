package index

import (
	"bytes"
	"io"
	"os"
	"sync"

	"github.com/meghashyamc/wheresthat/db/searchdb"
)

const maxContentExtractionSize = 5 * 1024 * 1024 // 5MB limit

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 64*1024)
	},
}

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
		doc.Content = string(content)
	}

	return doc, nil
}
func readTextContent(reader io.Reader, fileSize int64) ([]byte, error) {
	// Always cap the reader to avoid trusting fileSize blindly
	limitedReader := io.LimitReader(reader, maxContentExtractionSize)

	var buffer bytes.Buffer

	if fileSize > 0 && fileSize <= maxContentExtractionSize {
		buffer.Grow(int(fileSize))
	} else {
		buffer.Grow(maxContentExtractionSize)
	}

	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	for {
		n, err := limitedReader.Read(buf)
		if n > 0 {
			_, werr := buffer.Write(buf[:n])
			if werr != nil {
				return nil, werr
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}
