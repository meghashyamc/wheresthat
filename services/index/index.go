package index

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
)

type Service struct {
	logger   logger.Logger
	searchdb searchdb.DB
	kvDB     kvdb.DB
}

func New(logger logger.Logger, searchdb searchdb.DB, kvDB kvdb.DB) *Service {
	return &Service{
		logger:   logger,
		searchdb: searchdb,
		kvDB:     kvDB,
	}
}

func (s *Service) Create(rootPath string) error {
	// Get last index time to determine if this is incremental
	lastIndexTime, err := s.getLastIndexTime()
	if err != nil {
		s.logger.Error("could not get last index time", "err", err.Error())
		return err
	}

	files, err := s.getFilesToIndex(rootPath, lastIndexTime)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		s.logger.Info("no files to index")
		return nil
	}

	s.logger.Info("processing files...")
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

	s.logger.Info("building search index...")
	if err := s.searchdb.BuildIndex(documents); err != nil {
		s.logger.Error("failed to build index", "err", err.Error())
		return fmt.Errorf("failed to build search index: %w", err)
	}

	s.logger.Info("updating file metadata...")
	var metadataErrors []string
	for _, file := range files {
		metadata := kvdb.FileMetadata{
			LastIndexed: indexTime,
		}
		if err := s.setFileMetadata(file.Path, metadata); err != nil {
			s.logger.Error("failed to update file metadata", "path", file.Path, "err", err.Error())
			metadataErrors = append(metadataErrors, fmt.Sprintf("failed to update metadata for %s: %v", file.Path, err))
		}
	}

	if err := s.setLastIndexTime(indexTime); err != nil {
		s.logger.Error("failed to update last index time", "err", err.Error())
		return fmt.Errorf("failed to update last index time: %w", err)
	}

	if len(metadataErrors) > 0 {
		s.logger.Warn("some metadata updates failed", "errors", len(metadataErrors))
		for _, errMsg := range metadataErrors {
			s.logger.Warn(errMsg)
		}
	}

	s.logger.Info("index built successfully!", "indexed_files", len(files))
	return nil
}

func (s *Service) getFilesToIndex(rootPath string, lastIndexTime time.Time) ([]FileInfo, error) {

	s.logger.Info("performing incremental indexing", "last_index_time", lastIndexTime)
	files, err := s.discoverModifiedFiles(rootPath, lastIndexTime)
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

	return s.kvDB.Set(filepath, string(data))
}

func (s *Service) getFileMetadata(filepath string) (*kvdb.FileMetadata, error) {

	value, err := s.kvDB.Get(filepath)
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

func (s *Service) setLastIndexTime(indexTime time.Time) error {
	timeBytes, err := indexTime.MarshalBinary()
	if err != nil {
		s.logger.Error("failed to marshal index time", "time", indexTime, "err", err.Error())
		return fmt.Errorf("failed to marshal index time: %w", err)
	}

	return s.kvDB.Set(kvdb.LastIndexTimeKey, string(timeBytes))
}

func (s *Service) getLastIndexTime() (time.Time, error) {
	value, err := s.kvDB.Get(kvdb.LastIndexTimeKey)
	if err != nil {
		// Return zero time if not found (first time indexing)
		return time.Time{}, nil
	}

	var indexTime time.Time
	if err := indexTime.UnmarshalBinary([]byte(value)); err != nil {
		s.logger.Error("failed to unmarshal index time", "err", err.Error())
		return time.Time{}, fmt.Errorf("failed to unmarshal index time: %w", err)
	}

	return indexTime, nil
}
