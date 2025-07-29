package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
)

// Indexer represents the search database operations needed for index creation
type Indexer interface {
	BuildIndex(documents []searchdb.Document) error
	DeleteDocuments(documentIDs []string) error
	Close() error
}

const (
	ProgressStatusStep1    = 25
	ProgressStatusStep2    = 50
	ProgressStatusStep3    = 75
	ProgressStatusComplete = 100

	requestKeyPrefix = "request:"
	fileKeyPrefix    = "file:"
)

type Service struct {
	logger        logger.Logger
	indexer       Indexer
	metadataStore MetadataStore
	buildIndexC   chan indexRequest
}

type indexRequest struct {
	rootPath  string
	requestID string
}

func New(ctx context.Context, logger logger.Logger, indexer Indexer, metadataStore MetadataStore) *Service {
	indexService := &Service{
		logger:        logger,
		indexer:       indexer,
		metadataStore: metadataStore,
		buildIndexC:   make(chan indexRequest, 100),
	}

	go indexService.build(ctx)
	return indexService
}

// Create builds an index or incrementally updates it if it already exists
func (s *Service) Build(rootPath string, requestID string) error {
	// Initialize request status to 0
	if err := s.setRequestStatus(requestID, 0); err != nil {
		return fmt.Errorf("failed to initialize request status: %w", err)
	}

	s.buildIndexC <- indexRequest{
		rootPath:  rootPath,
		requestID: requestID,
	}
	return nil
}

// GetStatus retrieves the progress status for index creation
func (s *Service) GetStatus(requestID string) (int, error) {
	key := requestKeyPrefix + requestID
	value, err := s.metadataStore.Get(key)
	if err != nil {
		return 0, fmt.Errorf("request not found: %w", err)
	}

	status, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid status value: %w", err)
	}

	return status, nil
}

func (s *Service) build(ctx context.Context) {

	if r := recover(); r != nil {
		s.logger.Error("index service faced an unexpected issue", "err", r)
		s.build(ctx)
	}

	for {
		select {
		case req := <-s.buildIndexC:
			if err := s.buildIndex(req.rootPath, req.requestID); err != nil {
				s.logger.Error("failed to create index", "request_id", req.requestID, "err", err.Error())
			}
		case <-ctx.Done():
			s.logger.Info("index service stopped")
			return
		}
	}

}

func (s *Service) buildIndex(rootPath string, requestID string) error {
	files, err := s.getFilesToIndex(rootPath)
	if err != nil {
		return err
	}

	// Update progress to ProgressStatusStep1% after getFilesToIndex completes
	if err := s.setRequestStatus(requestID, ProgressStatusStep1); err != nil {
		s.logger.Error("failed to update request status", "requestID", requestID, "progress", ProgressStatusStep1, "err", err.Error())
	}

	// Identify and remove deleted files before indexing new/modified files
	deletedFiles, err := s.getDeletedFiles()
	if err != nil {
		return err
	}

	if err := s.removeDeletedFiles(deletedFiles); err != nil {
		return err
	}

	// Update progress to ProgressStatusStep2% after getDeletedFiles and removeDeletedFiles complete
	if err := s.setRequestStatus(requestID, ProgressStatusStep2); err != nil {
		s.logger.Error("failed to update request status", "requestID", requestID, "progress", ProgressStatusStep2, "err", err.Error())
	}

	if len(files) == 0 {
		s.logger.Info("no files to index")
		// Mark as complete if no files to index
		if err := s.setRequestStatus(requestID, ProgressStatusComplete); err != nil {
			s.logger.Error("failed to update request status", "requestID", requestID, "progress", 100, "err", err.Error())
		}
		return nil
	}

	return s.doBuildIndex(files, requestID)
}

func (s *Service) removeDeletedFiles(deletedFiles []string) error {
	if len(deletedFiles) == 0 {
		return nil
	}
	s.logger.Info("removing deleted files from index", "deleted_files", len(deletedFiles))
	if err := s.indexer.DeleteDocuments(deletedFiles); err != nil {
		s.logger.Error("failed to delete documents from search index", "err", err.Error())
		return fmt.Errorf("failed to delete documents from search index: %w", err)
	}

	// Remove metadata for deleted files
	for _, filePath := range deletedFiles {
		if err := s.metadataStore.Delete(fileKeyPrefix + filePath); err != nil {
			s.logger.Error("failed to delete file metadata", "path", filePath, "err", err.Error())
		}
	}
	return nil
}

func (s *Service) doBuildIndex(files []FileInfo, requestID string) error {
	s.logger.Info("building index of files...")
	var documents []searchdb.Document
	indexTime := time.Now().UTC()

	for i, file := range files {
		if i%100 == 0 {
			s.logger.Info(fmt.Sprintf("processed %d/%d files", i, len(files)))
		}

		doc, err := extractContent(file)
		if err != nil {
			s.logger.Error("error processing file", "path", file.Path, "err", err.Error())
			continue
		}

		documents = append(documents, *doc)
	}

	// Update progress to ProgressStatusStep3% after for loop completes but before BuildIndex
	if err := s.setRequestStatus(requestID, ProgressStatusStep3); err != nil {
		s.logger.Error("failed to update request status", "requestID", requestID, "progress", ProgressStatusStep3, "err", err.Error())
	}

	s.logger.Info("building search index...")
	if err := s.indexer.BuildIndex(documents); err != nil {
		s.logger.Error("failed to build index", "err", err.Error())
		return fmt.Errorf("failed to build search index: %w", err)
	}

	s.logger.Info("updating file metadata...")
	for _, file := range files {
		metadata := kvdb.FileMetadata{
			LastIndexed: indexTime,
		}
		if err := s.setFileMetadata(file.Path, metadata); err != nil {
			s.logger.Error("failed to update file metadata", "path", file.Path, "err", err.Error())
			return fmt.Errorf("failed to update file metadata: %w", err)
		}
	}

	// Update progress to 100% after index building completes
	if err := s.setRequestStatus(requestID, ProgressStatusComplete); err != nil {
		s.logger.Error("failed to update request status", "requestID", requestID, "progress", 100, "err", err.Error())
	}

	s.logger.Info("index built successfully!", "indexed_files", len(files))
	return nil
}

func (s *Service) getFilesToIndex(rootPath string) ([]FileInfo, error) {

	s.logger.Info("performing incremental indexing")
	files, err := s.discoverModifiedFiles(rootPath)
	if err != nil {
		s.logger.Error("could not discover modified files", "err", err.Error())
		return nil, err
	}
	s.logger.Info("discovered modified files", slog.Int("num_of_files", len(files)))

	return files, nil
}

func (s *Service) setFileMetadata(filepath string, metadata kvdb.FileMetadata) error {
	if filepath == "" {
		s.logger.Error("filepath cannot be empty", "filepath", filepath)
		return fmt.Errorf("filepath cannot be empty")
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		s.logger.Error("failed to marshal metadata", "filepath", filepath, "err", err.Error())
		return fmt.Errorf("failed to marshal metadata for %s: %w", filepath, err)
	}

	return s.metadataStore.Set(fileKeyPrefix+filepath, string(data))
}

func (s *Service) getFileMetadata(filepath string) (*kvdb.FileMetadata, error) {

	value, err := s.metadataStore.Get(fileKeyPrefix + filepath)
	if err != nil {
		return nil, err
	}

	var metadata kvdb.FileMetadata
	if err := json.Unmarshal([]byte(value), &metadata); err != nil {
		s.logger.Error("failed to unmarshal metadata", "filepath", filepath, "err", err.Error())
		return nil, fmt.Errorf("failed to unmarshal metadata for %s: %w", filepath, err)
	}

	return &metadata, nil
}

func (s *Service) getDeletedFiles() ([]string, error) {
	allKeys, err := s.metadataStore.GetAllKeys()
	if err != nil {
		s.logger.Error("failed to get all keys from database", "err", err.Error())
		return nil, fmt.Errorf("failed to get all keys from database: %w", err)
	}

	var deletedFiles []string
	for _, key := range allKeys {
		// Skip non-file keys
		if !strings.HasPrefix(key, fileKeyPrefix) {
			continue
		}

		filePath := strings.TrimPrefix(key, fileKeyPrefix)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			deletedFiles = append(deletedFiles, filePath)
		}
	}

	return deletedFiles, nil
}

func (s *Service) setRequestStatus(requestID string, status int) error {
	key := requestKeyPrefix + requestID
	return s.metadataStore.Set(key, strconv.Itoa(status))
}
