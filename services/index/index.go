package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
)

// Indexer represents the search database operations needed for index creation
type Indexer interface {
	BuildIndex(documents []*searchdb.Document) error
	DeleteDocuments(documentIDs []string) error
	Close() error
}

const (
	ProgressStatusStep1    = 10
	ProgressStatusStep2    = 20
	ProgressStatusComplete = 100
	ProgressStatusFailed   = -1

	maxGoRoutinesForFileProcessing = 50
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
	s.setRequestStatus(requestID, 0)

	// Leads to s.buildIndex being called
	s.buildIndexC <- indexRequest{
		rootPath:  rootPath,
		requestID: requestID,
	}
	return nil
}

// GetStatus retrieves the progress status for index creation
func (s *Service) GetStatus(requestID string) (int, error) {
	value, err := s.metadataStore.Get(kvdb.RequestsBucket, requestID)
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
			s.buildIndex(req.rootPath, req.requestID)
		case <-ctx.Done():
			s.logger.Info("index service stopped")
			return
		}
	}

}

func (s *Service) buildIndex(rootPath string, requestID string) {
	files, err := s.getFilesToIndex(rootPath)
	if err != nil {
		s.logger.Error("failed to create index", "request_id", requestID, "err", err.Error())
		s.setRequestStatus(requestID, ProgressStatusFailed)
	}

	// Update progress to ProgressStatusStep1% after getFilesToIndex completes
	s.setRequestStatus(requestID, ProgressStatusStep1)

	// Identify and remove deleted files before indexing new/modified files
	deletedFiles, err := s.getDeletedFiles()
	if err != nil {
		s.logger.Error("failed to create index", "request_id", requestID, "err", err.Error())
		s.setRequestStatus(requestID, ProgressStatusFailed)
	}

	if err := s.removeDeletedFiles(deletedFiles); err != nil {
		s.logger.Error("failed to create index", "request_id", requestID, "err", err.Error())
		s.setRequestStatus(requestID, ProgressStatusFailed)
	}

	// Update progress to ProgressStatusStep2% after getDeletedFiles and removeDeletedFiles complete
	s.setRequestStatus(requestID, ProgressStatusStep2)

	s.doBuildIndex(files, requestID)

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
		if err := s.metadataStore.Delete(kvdb.FilesBucket, filePath); err != nil {
			s.logger.Error("failed to delete file metadata", "path", filePath, "err", err.Error())
		}
	}
	return nil
}

func (s *Service) doBuildIndex(files []FileInfo, requestID string) {
	s.logger.Info("building index of files...")
	indexTime := time.Now().UTC()

	if len(files) == 0 {
		s.setRequestStatus(requestID, ProgressStatusComplete)
		s.logger.Info("no files to index")
		return
	}

	numGoroutines := min(maxGoRoutinesForFileProcessing, len(files))
	filesPerGoroutine := len(files) / numGoroutines
	if filesPerGoroutine == 0 {
		filesPerGoroutine = 1
	}

	// Channel to collect processed files for metadata updates
	processedFilesChan := make(chan []FileInfo, numGoroutines)

	// WaitGroup to wait for all goroutines to complete
	var indexWG sync.WaitGroup

	s.logger.Info("starting parallel indexing", "goroutines", numGoroutines, "files_per_goroutine", filesPerGoroutine)

	for i := range numGoroutines {
		start := i * filesPerGoroutine
		end := start + filesPerGoroutine

		// For the last goroutine, include any remaining files
		if i == numGoroutines-1 {
			end = len(files)
		}
		if start >= len(files) {
			break
		}

		indexWG.Add(1)
		go s.doBuildIndexForFilesPortion(files[start:end], i, processedFilesChan, &indexWG)
	}

	var metadataWG sync.WaitGroup
	metadataWG.Add(1)

	// This is primarily so that future index requests don't lead to reindexing files that
	// are already indexed
	go s.updateMetadata(indexTime, requestID, len(files), processedFilesChan, &metadataWG)

	go func() {
		indexWG.Wait()
		close(processedFilesChan)
	}()

	metadataWG.Wait()

	// Update progress to 100% after index building and metadata updation completes
	s.setRequestStatus(requestID, ProgressStatusComplete)

}

func (s *Service) updateMetadata(indexTime time.Time, requestID string, totalFilesCount int, processedFilesChan chan []FileInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	s.logger.Info("updating file metadata...")
	updatedCount := 0

	for processedFiles := range processedFilesChan {
		for _, file := range processedFiles {
			metadata := kvdb.FileMetadata{
				LastIndexed: indexTime,
			}
			if err := s.setFileMetadata(file.Path, metadata); err == nil {
				updatedCount++
			}

		}
		if updatedCount%1000 == 0 {
			s.logger.Info("updated metadata for files:", "count", fmt.Sprintf("%d/%d", updatedCount, totalFilesCount))
			status := getProgressPercentage(updatedCount, totalFilesCount, ProgressStatusStep2, ProgressStatusComplete)
			s.setRequestStatus(requestID, status)
		}
	}

	s.logger.Info("finished updating metadata successfully!", "count", fmt.Sprintf("%d/%d", updatedCount, totalFilesCount))

}

func (s *Service) getFilesToIndex(rootPath string) ([]FileInfo, error) {

	s.logger.Info("performing incremental indexing")
	files, err := s.discoverModifiedFiles(rootPath)
	if err != nil {
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

	if err := s.metadataStore.Set(kvdb.FilesBucket, filepath, string(data)); err != nil {
		s.logger.Error("failed to set file metadata", "filepath", filepath, "err", err.Error())
		return err
	}

	return nil
}

func (s *Service) getFileMetadata(filepath string) (*kvdb.FileMetadata, error) {

	value, err := s.metadataStore.Get(kvdb.FilesBucket, filepath)
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
	allKeys, err := s.metadataStore.GetAllKeys(kvdb.FilesBucket)
	if err != nil {
		s.logger.Error("failed to get all keys from database", "err", err.Error())
		return nil, fmt.Errorf("failed to get all keys from database: %w", err)
	}

	var deletedFiles []string
	for _, key := range allKeys {

		if _, err := os.Stat(key); os.IsNotExist(err) {
			deletedFiles = append(deletedFiles, key)
		}
	}

	return deletedFiles, nil
}

func (s *Service) setRequestStatus(requestID string, status int) {
	if err := s.metadataStore.Set(kvdb.RequestsBucket, requestID, strconv.Itoa(status)); err != nil {
		s.logger.Error("failed to update request status", "requestID", requestID, "progress", ProgressStatusStep1, "err", err.Error())
	}
}

func (s *Service) doBuildIndexForFilesPortion(filesPortion []FileInfo, goroutineID int, processedFilesChan chan []FileInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	numOfFiles := len(filesPortion)
	totalProcessedFilesCount := 0
	for i := 0; i <= max(0, numOfFiles-searchdb.IndexingBatchSize); i += searchdb.IndexingBatchSize {

		processedFiles := s.doBuildIndexForSingleBatchOfFiles(filesPortion[i:min(i+searchdb.IndexingBatchSize, numOfFiles)], goroutineID)
		totalProcessedFilesCount += len(processedFiles)
		processedFilesChan <- processedFiles

		s.logger.Info(fmt.Sprintf("goroutine %d processed %d/%d files", goroutineID, totalProcessedFilesCount, numOfFiles))
	}
	s.logger.Info("completed indexing for goroutine", "goroutine_id", goroutineID, "num_of_files_received", numOfFiles)

}

func (s *Service) doBuildIndexForSingleBatchOfFiles(filesInBatch []FileInfo, goroutineID int) []FileInfo {

	var documents []*searchdb.Document
	var processedFiles []FileInfo

	for _, file := range filesInBatch {

		doc, err := extractContent(file)
		if err != nil {
			s.logger.Error("error processing file", "path", file.Path, "err", err.Error(), "go_routine_id", goroutineID)
			continue
		}
		documents = append(documents, doc)
		processedFiles = append(processedFiles, file)
	}

	if err := s.indexer.BuildIndex(documents); err != nil {
		s.logger.Error("failed to build index for goroutine", "goroutine_id", goroutineID, "err", err.Error())
		return make([]FileInfo, 0)
	}

	return processedFiles

}

func getProgressPercentage(done int, total int, initial int, final int) int {
	if done == 0 || total == 0 {
		return initial
	}

	if done >= total {
		return final
	}

	// Calculate the percentage between initial and final
	progress := float64(done) / float64(total)
	result := float64(initial) + progress*float64(final-initial)

	return int(result)

}
