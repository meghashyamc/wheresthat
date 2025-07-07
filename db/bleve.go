package db

import (
	"fmt"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/meghashyamc/wheresthat/logger"
)

const indexingBatchSize = 100

type BleveDB struct {
	logger logger.Logger
}

type SearchDocument struct {
	ID      string    `json:"id"`
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Content string    `json:"content"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Type    string    `json:"type"`
}

func New(logger logger.Logger) *BleveDB {
	return &BleveDB{logger: logger}
}

func (b *BleveDB) BuildIndex(documents any, indexPath string) error {
	docs, ok := documents.([]SearchDocument)
	if !ok {
		return fmt.Errorf("documents must be of type []SearchDocument")
	}
	// Create index mapping
	mapping := createIndexMapping()

	// Create or open index
	index, err := bleve.New(indexPath, mapping)
	if err != nil {
		// If index exists, try to open it
		index, err = bleve.Open(indexPath)
		if err != nil {
			b.logger.Error("could not open index", "err", err.Error())
			return err
		}
	}

	// Index documents in batches for better performance
	batch := index.NewBatch()

	for i, doc := range docs {

		err := batch.Index(doc.ID, doc)
		if err != nil {
			b.logger.Error("could not index document", "err", err.Error())
			return err
		}

		// Execute batch when it reaches the batch size
		if (i+1)%indexingBatchSize == 0 {
			err = index.Batch(batch)
			if err != nil {
				return err
			}
			batch = index.NewBatch()
		}
	}

	// Execute remaining documents
	if batch.Size() > 0 {
		err = index.Batch(batch)
		if err != nil {
			b.logger.Error("could not index document", "err", err.Error())
			return err
		}
	}

	return nil
}

func createIndexMapping() mapping.IndexMapping {
	// Create a mapping for our documents
	indexMapping := bleve.NewIndexMapping()

	// Create document mapping
	docMapping := bleve.NewDocumentMapping()

	// Path field - not analyzed (exact match)
	pathFieldMapping := bleve.NewTextFieldMapping()
	pathFieldMapping.Analyzer = "keyword"
	docMapping.AddFieldMappingsAt("path", pathFieldMapping)

	// Name field - analyzed for partial matching
	nameFieldMapping := bleve.NewTextFieldMapping()
	nameFieldMapping.Analyzer = "standard"
	docMapping.AddFieldMappingsAt("name", nameFieldMapping)

	// Content field - analyzed for full-text search
	contentFieldMapping := bleve.NewTextFieldMapping()
	contentFieldMapping.Analyzer = "standard"
	contentFieldMapping.Store = false // Don't store full content in index
	contentFieldMapping.Index = true  // But do index it for searching
	docMapping.AddFieldMappingsAt("content", contentFieldMapping)

	// Size field - numeric
	sizeFieldMapping := bleve.NewNumericFieldMapping()
	docMapping.AddFieldMappingsAt("size", sizeFieldMapping)

	// Type field - not analyzed
	typeFieldMapping := bleve.NewTextFieldMapping()
	typeFieldMapping.Analyzer = "keyword"
	docMapping.AddFieldMappingsAt("type", typeFieldMapping)

	indexMapping.AddDocumentMapping("_default", docMapping)

	return indexMapping
}
