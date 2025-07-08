package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/meghashyamc/wheresthat/logger"
)

const indexingBatchSize = 100
const indexPath = "./search.index"

type BleveDB struct {
	logger logger.Logger
}

func New(logger logger.Logger) *BleveDB {
	return &BleveDB{logger: logger}
}

func (b *BleveDB) BuildIndex(documents []Document) error {

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

	for i, doc := range documents {

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

func (b *BleveDB) Search(queryString string, limit int, offset int) (*SearchResponse, error) {
	start := time.Now()

	// Open the index
	index, err := bleve.Open(indexPath)
	if err != nil {
		b.logger.Error("could not open index for search", "err", err.Error(), "path", indexPath)
		return nil, fmt.Errorf("could not open search index: %w", err)
	}
	defer index.Close()

	// Create search query
	searchQuery := b.buildSearchQuery(queryString)

	// Create search request
	searchRequest := bleve.NewSearchRequestOptions(searchQuery, limit, offset, false)

	// Enable highlighting for content matches
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Highlight.AddField("content")
	searchRequest.Highlight.AddField("name")
	searchRequest.Fields = []string{"path", "name", "size", "mod_time", "type"}

	// Execute search
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		b.logger.Error("search failed", "err", err.Error())
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results
	results := make([]SearchResult, len(searchResult.Hits))
	for i, hit := range searchResult.Hits {
		result := SearchResult{
			ID:       hit.ID,
			Score:    hit.Score,
			Fields:   make(map[string]string),
			Snippets: make([]string, 0),
		}

		// Extract fields
		if path, ok := hit.Fields["path"].(string); ok {
			result.Path = path
		}
		if name, ok := hit.Fields["name"].(string); ok {
			result.Name = name
		}
		if size, ok := hit.Fields["size"].(float64); ok {
			result.Size = int64(size)
		}
		if modTime, ok := hit.Fields["mod_time"].(string); ok {
			result.ModTime = modTime
		}
		if fileType, ok := hit.Fields["type"].(string); ok {
			result.Type = fileType
		}

		// Extract highlights/snippets
		if hit.Fragments != nil {
			for field, fragments := range hit.Fragments {
				result.Fields[field] = strings.Join(fragments, " ... ")
				result.Snippets = append(result.Snippets, fragments...)
			}
		}

		results[i] = result
	}

	searchTime := time.Since(start)

	response := &SearchResponse{
		Results:    results,
		Total:      searchResult.Total,
		MaxScore:   searchResult.MaxScore,
		SearchTime: searchTime.String(),
	}

	return response, nil
}

func (b *BleveDB) buildSearchQuery(queryString string) query.Query {
	// Trim whitespace
	queryString = strings.TrimSpace(queryString)

	if queryString == "" {
		return bleve.NewMatchAllQuery()
	}

	// Create a disjunction query that searches across multiple fields
	disjunctQuery := bleve.NewDisjunctionQuery()

	// Search in content (highest boost)
	contentQuery := bleve.NewMatchQuery(queryString)
	contentQuery.SetField("content")
	contentQuery.SetBoost(3.0)
	disjunctQuery.AddQuery(contentQuery)

	// Search in file name (medium boost)
	nameQuery := bleve.NewMatchQuery(queryString)
	nameQuery.SetField("name")
	nameQuery.SetBoost(2.0)
	disjunctQuery.AddQuery(nameQuery)

	// Search in path (lower boost)
	pathQuery := bleve.NewMatchQuery(queryString)
	pathQuery.SetField("path")
	pathQuery.SetBoost(1.0)
	disjunctQuery.AddQuery(pathQuery)

	// Add phrase search for exact matches (highest boost)
	phraseQuery := bleve.NewMatchPhraseQuery(queryString)
	phraseQuery.SetField("content")
	phraseQuery.SetBoost(5.0)
	disjunctQuery.AddQuery(phraseQuery)

	// Add prefix search for partial matches
	if len(queryString) > 2 {
		prefixQuery := bleve.NewPrefixQuery(queryString)
		prefixQuery.SetField("name")
		prefixQuery.SetBoost(1.5)
		disjunctQuery.AddQuery(prefixQuery)
	}

	return disjunctQuery
}
