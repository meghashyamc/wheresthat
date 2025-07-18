package searchdb

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/meghashyamc/wheresthat/config"
	"github.com/meghashyamc/wheresthat/logger"
)

const indexingBatchSize = 100
const snippetContext = 100

const (
	indexFieldContent = "content"
	indexFieldName    = "name"
	indexFieldPath    = "path"
	indexFieldSize    = "size"
	indexFieldModTime = "mod_time"
)

type BleveDB struct {
	indexPath string
	logger    logger.Logger
	index     bleve.Index
}

func New(logger logger.Logger, cfg *config.Config) (*BleveDB, error) {
	mapping := createIndexMapping()
	indexPath := filepath.Join(cfg.GetStoragePath(), cfg.GetIndexPath())
	index, err := bleve.New(indexPath, mapping)
	if err != nil {
		index, err = bleve.Open(indexPath)
		if err != nil {
			logger.Error("could not open index", "err", err.Error())
			return nil, err
		}
	}
	return &BleveDB{indexPath: indexPath, logger: logger, index: index}, nil
}

func (b *BleveDB) BuildIndex(documents []Document) error {

	batch := b.index.NewBatch()

	for i, doc := range documents {

		err := batch.Index(doc.ID, doc)
		if err != nil {
			b.logger.Error("could not index document", "err", err.Error())
			return err
		}

		// Execute batch when it reaches the batch size
		if (i+1)%indexingBatchSize == 0 {
			err = b.index.Batch(batch)
			if err != nil {
				return err
			}
			batch = b.index.NewBatch()
		}
	}

	if batch.Size() > 0 {
		if err := b.index.Batch(batch); err != nil {
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

func (b *BleveDB) Search(queryString string, limit int, offset int) (*Response, error) {
	start := time.Now()

	searchQuery := b.buildSearchQuery(queryString)

	searchRequest := bleve.NewSearchRequestOptions(searchQuery, limit, offset, false)

	searchRequest.Fields = []string{indexFieldPath, indexFieldName, indexFieldSize, indexFieldModTime}

	// Enable highlighting for content field
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Highlight.AddField(indexFieldContent)

	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		b.logger.Error("search failed", "err", err.Error())
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]Result, len(searchResult.Hits))
	for i, hit := range searchResult.Hits {
		result := Result{
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

		// Extract snippet if content matches exist
		result.Snippet = b.extractSnippet(result.Path, hit.Locations, queryString)

		results[i] = result
	}

	searchTime := time.Since(start)

	response := &Response{
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

func (b *BleveDB) DeleteDocuments(documentIDs []string) error {
	batch := b.index.NewBatch()

	for i, docID := range documentIDs {
		batch.Delete(docID)

		// Execute batch when it reaches the batch size
		if (i+1)%indexingBatchSize == 0 {
			err := b.index.Batch(batch)
			if err != nil {
				return err
			}
			batch = b.index.NewBatch()
		}
	}

	if batch.Size() > 0 {
		if err := b.index.Batch(batch); err != nil {
			b.logger.Error("could not delete documents", "err", err.Error())
			return err
		}
	}

	return nil
}

func (b *BleveDB) GetDocCount() (uint64, error) {
	return b.index.DocCount()
}

func (b *BleveDB) Close() error {

	if b.index != nil {
		if err := b.index.Close(); err != nil {
			b.logger.Error("could not close search index", "err", err.Error())
			return err
		}
	}
	return nil
}

func (b *BleveDB) extractSnippet(filePath string, locations search.FieldTermLocationMap, queryString string) string {
	// Check if there are content locations from the search
	contentLocations, hasContentMatch := locations[indexFieldContent]

	if !hasContentMatch || len(contentLocations) == 0 {
		return ""
	}

	// Check if file is text-based
	if !b.isTextFile(filePath) {
		return ""
	}

	// Try to read the file and extract snippet from the first location
	snippet, err := b.readSnippetFromLocation(filePath, contentLocations)
	if err != nil {
		b.logger.Warn("failed to extract snippet from file", "path", filePath, "err", err.Error())
		return ""
	}

	return snippet
}

func (b *BleveDB) isTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Common text file extensions
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".js": true, ".py": true,
		".java": true, ".cpp": true, ".c": true, ".h": true, ".css": true,
		".html": true, ".htm": true, ".xml": true, ".json": true, ".yaml": true,
		".yml": true, ".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".sql": true, ".log": true, ".conf": true, ".cfg": true, ".ini": true,
		".toml": true, ".rs": true, ".rb": true, ".php": true, ".pl": true,
		".swift": true, ".kt": true, ".scala": true, ".clj": true, ".hs": true,
		".ml": true, ".elm": true, ".r": true, ".m": true, ".tex": true,
		".dockerfile": true, ".makefile": true, ".cmake": true, ".gradle": true,
		".maven": true, ".sbt": true, ".lock": true, ".env": true, ".gitignore": true,
		".gitattributes": true, ".editorconfig": true, ".prettierrc": true,
		".eslintrc": true, ".babelrc": true, ".nvmrc": true, ".nodeversion": true,
	}

	if textExtensions[ext] {
		return true
	}

	// Check MIME type as fallback
	mimeType := mime.TypeByExtension(ext)
	return strings.HasPrefix(mimeType, "text/")
}

func (b *BleveDB) readSnippetFromLocation(filePath string, termLocations search.TermLocationMap) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		b.logger.Error("failed to open file for snippet", "path", filePath, "err", err.Error())
		return "", err
	}
	defer file.Close()

	// Get file size to avoid reading beyond the file
	fileInfo, err := file.Stat()
	if err != nil {
		b.logger.Error("failed to get file info for snippet", "path", filePath, "err", err.Error())
		return "", err
	}
	fileSize := fileInfo.Size()

	var matchStart, matchEnd uint64
	found := false

	// Look through all terms to find the first location
	for _, locations := range termLocations {
		if len(locations) > 0 && locations[0] != nil {
			matchStart = locations[0].Start
			matchEnd = locations[0].End
			found = true
			break
		}
	}

	if !found {
		b.logger.Error("no match found for snippet", "path", filePath)
		return "", nil
	}

	if matchStart >= uint64(fileSize) {
		b.logger.Error("match start is beyond file size for snippet", "path", filePath, "matchStart", matchStart, "fileSize", fileSize)
		return "", nil
	}
	if matchEnd > uint64(fileSize) {
		b.logger.Error("match end is beyond file size for snippet", "path", filePath, "matchEnd", matchEnd, "fileSize", fileSize)
		matchEnd = uint64(fileSize)
	}

	// Calculate snippet boundaries with context
	snippetStart := max(0, int64(matchStart)-int64(snippetContext))
	snippetEnd := min(fileSize, int64(matchEnd)+int64(snippetContext))

	// Calculate buffer size needed
	bufferSize := snippetEnd - snippetStart
	if bufferSize <= 0 {
		b.logger.Error("invalid buffer size for snippet", "path", filePath, "snippetStart", snippetStart, "snippetEnd", snippetEnd)
		return "", nil
	}

	// Read only the snippet portion using ReadAt
	buffer := make([]byte, bufferSize)
	_, err = file.ReadAt(buffer, snippetStart)
	if err != nil && err != io.EOF {
		b.logger.Error("failed to read file for snippet", "path", filePath, "err", err.Error())
		return "", err
	}

	return formatSnippet(string(buffer), snippetStart, snippetEnd, fileSize), nil
}

func formatSnippet(snippet string, snippetStart int64, snippetEnd int64, fileSize int64) string {
	snippet = strings.TrimSpace(snippet)
	if snippetStart > 0 {
		snippet = "..." + snippet
	}
	if snippetEnd < fileSize {
		snippet = snippet + "..."
	}

	return snippet
}
