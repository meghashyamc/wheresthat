package index

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/meghashyamc/wheresthat/db"
	"github.com/meghashyamc/wheresthat/logger"
)

type SearchDocument struct {
	ID      string    `json:"id"`
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Content string    `json:"content"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Type    string    `json:"type"`
}

type Service struct {
	logger logger.Logger
	db     db.DB
}

func New(logger logger.Logger, db db.DB) *Service {
	return &Service{
		logger: logger,
		db:     db,
	}
}

func (s *Service) Create(indexPath string) error {
	s.logger.Info("discovering files...")
	files, err := discoverFiles(indexPath)
	if err != nil {
		log.Fatal(err)
	}
	s.logger.Info("discovered files", slog.Int("num_of_files", len(files)))

	// Step 2: Extract content and process
	s.logger.Info("Processing files...")
	var documents []*Document
	for i, file := range files {
		if i%100 == 0 {
			s.logger.Info("Processed %d/%d files\n", i, len(files))
		}

		doc, err := extractContent(file)
		if err != nil {
			s.logger.Error("Error processing %s: %v\n", file.Path, err)
			continue
		}

		documents = append(documents, doc)
	}

	s.logger.Info("Building search index...")
	index, err := buildIndex(documents, "./search.index")
	if err != nil {
		log.Fatal(err)
	}

	docCount, _ := index.DocCount()
	s.logger.Info("Index built successfully!", slog.Uint64("num_of_documents", docCount))

	indexStat, _ := os.Stat("./search.index")
	s.logger.Info("Index size in bytes", slog.Int64("size", indexStat.Size()))

	return nil
}
func buildIndex(documents []*Document, indexPath string) (bleve.Index, error) {
	// Create index mapping
	mapping := createIndexMapping()

	// Create or open index
	index, err := bleve.New(indexPath, mapping)
	if err != nil {
		// If index exists, try to open it
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, err
		}
	}

	// Index documents in batches for better performance
	batch := index.NewBatch()
	batchSize := 100

	for i, doc := range documents {
		searchDoc := &SearchDocument{
			ID:      doc.ID,
			Path:    doc.Path,
			Name:    doc.Name,
			Content: doc.Content,
			Size:    doc.Size,
			ModTime: doc.ModTime,
			Type:    doc.Type,
		}

		err := batch.Index(doc.ID, searchDoc)
		if err != nil {
			return nil, err
		}

		// Execute batch when it reaches the batch size
		if (i+1)%batchSize == 0 {
			err = index.Batch(batch)
			if err != nil {
				return nil, err
			}
			batch = index.NewBatch()
		}
	}

	// Execute remaining documents
	if batch.Size() > 0 {
		err = index.Batch(batch)
		if err != nil {
			return nil, err
		}
	}

	return index, nil
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
