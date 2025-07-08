package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/meghashyamc/wheresthat/logger"
)

const indexingBatchSize = 100
const indexPath = "./search.index"
const (
	indexFieldContent = "content"
	indexFieldName    = "name"
	indexFieldPath    = "path"
	indexFieldSize    = "size"
	indexFieldModTime = "mod_time"
)

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
	defer index.Close()

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

	indexMapping := bleve.NewIndexMapping()
	docMapping := bleve.NewDocumentMapping()

	// Path field - not analyzed (exact match)
	pathFieldMapping := bleve.NewTextFieldMapping()
	pathFieldMapping.Analyzer = keyword.Name
	docMapping.AddFieldMappingsAt(indexFieldPath, pathFieldMapping)

	// Name field - analyzed for partial matching
	nameFieldMapping := bleve.NewTextFieldMapping()
	nameFieldMapping.Analyzer = standard.Name
	docMapping.AddFieldMappingsAt(indexFieldName, nameFieldMapping)

	// Content field - analyzed for full-text search
	contentFieldMapping := bleve.NewTextFieldMapping()
	contentFieldMapping.Analyzer = standard.Name
	contentFieldMapping.Store = false // Don't store full content in index
	contentFieldMapping.Index = true  // But do index it for searching
	docMapping.AddFieldMappingsAt(indexFieldContent, contentFieldMapping)

	sizeFieldMapping := bleve.NewNumericFieldMapping()
	docMapping.AddFieldMappingsAt(indexFieldSize, sizeFieldMapping)

	indexMapping.AddDocumentMapping("_default", docMapping)

	return indexMapping
}

func (b *BleveDB) Search(queryString string, limit int, offset int) (*SearchResponse, error) {
	start := time.Now()

	index, err := bleve.Open(indexPath)
	if err != nil {
		b.logger.Error("could not open index for search", "err", err.Error(), "path", indexPath)
		return nil, fmt.Errorf("could not open search index: %w", err)
	}
	defer index.Close()

	searchQuery := b.buildSearchQuery(queryString)

	searchRequest := bleve.NewSearchRequestOptions(searchQuery, limit, offset, false)

	searchRequest.Fields = []string{indexFieldPath, indexFieldName, indexFieldSize, indexFieldModTime}

	searchResult, err := index.Search(searchRequest)
	if err != nil {
		b.logger.Error("search failed", "err", err.Error())
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]SearchResult, len(searchResult.Hits))
	for i, hit := range searchResult.Hits {
		result := SearchResult{
			ID:    hit.ID,
			Score: hit.Score,
		}

		if path, ok := hit.Fields[indexFieldPath].(string); ok {
			result.Path = path
		}
		if name, ok := hit.Fields[indexFieldName].(string); ok {
			result.Name = name
		}
		if size, ok := hit.Fields[indexFieldSize].(float64); ok {
			result.Size = int64(size)
		}
		if modTime, ok := hit.Fields[indexFieldModTime].(string); ok {
			result.ModTime = modTime
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

	const (
		boostForContent      = 3.0
		boostForFileName     = 2.0
		boostForPath         = 1.0
		boostForPhraseMatch  = 5.0
		boostForPartialMatch = 1.5
	)

	queryString = strings.ToLower(strings.TrimSpace(queryString))

	if queryString == "" {
		return bleve.NewMatchAllQuery()
	}

	disjunctQuery := bleve.NewDisjunctionQuery()

	contentQuery := bleve.NewMatchQuery(queryString)
	contentQuery.SetField(indexFieldContent)
	contentQuery.SetBoost(boostForContent)
	disjunctQuery.AddQuery(contentQuery)

	nameQuery := bleve.NewMatchQuery(queryString)
	nameQuery.SetField(indexFieldName)
	nameQuery.SetBoost(boostForFileName)
	disjunctQuery.AddQuery(nameQuery)

	pathQuery := bleve.NewMatchQuery(queryString)
	pathQuery.SetField(indexFieldPath)
	pathQuery.SetBoost(boostForPath)
	disjunctQuery.AddQuery(pathQuery)

	phraseQuery := bleve.NewMatchPhraseQuery(queryString)
	phraseQuery.SetField(indexFieldContent)
	phraseQuery.SetBoost(boostForPhraseMatch)
	disjunctQuery.AddQuery(phraseQuery)

	if len(queryString) > 2 {
		prefixQuery := bleve.NewPrefixQuery(queryString)
		prefixQuery.SetField(indexFieldName)
		prefixQuery.SetBoost(boostForPartialMatch)
		disjunctQuery.AddQuery(prefixQuery)

		contentPrefixQuery := bleve.NewPrefixQuery(queryString)
		contentPrefixQuery.SetField(indexFieldContent)
		contentPrefixQuery.SetBoost(boostForPartialMatch)
		disjunctQuery.AddQuery(contentPrefixQuery)
	}

	return disjunctQuery
}
