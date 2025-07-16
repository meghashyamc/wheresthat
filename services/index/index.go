package index

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
)

type Service struct {
	logger   logger.Logger
	searchDB searchdb.DB
	kvDB     kvdb.DB
}

func New(logger logger.Logger, searchDB searchdb.DB, kvDB kvdb.DB) *Service {
	return &Service{
		logger:   logger,
		searchDB: searchDB,
		kvDB:     kvDB,
	}
}

func (s *Service) Create(rootPath string) error {

	files, err := s.getFilesToIndex(rootPath)
	if err != nil {
		return err
	}

	// Identify and remove deleted files before indexing new/modified files
	deletedFiles, err := s.getDeletedFiles()
	if err != nil {
		return err
	}

	if err := s.removeDeletedFiles(deletedFiles); err != nil {
		return err
	}

	if len(files) == 0 {
		s.logger.Info("no files to index")
		return nil
	}

	return s.buildIndex(files)
}

func (s *Service) removeDeletedFiles(deletedFiles []string) error {
	if len(deletedFiles) == 0 {
		return nil
	}
	s.logger.Info("removing deleted files from index", "deleted_files", len(deletedFiles))
	if err := s.searchDB.DeleteDocuments(deletedFiles); err != nil {
		s.logger.Error("failed to delete documents from search index", "err", err.Error())
		return fmt.Errorf("failed to delete documents from search index: %w", err)
	}

	// Remove metadata for deleted files
	for _, filePath := range deletedFiles {
		if err := s.kvDB.Delete(filePath); err != nil {
			s.logger.Error("failed to delete file metadata", "path", filePath, "err", err.Error())
		}
	}
	return nil
}

func (s *Service) buildIndex(files []FileInfo) error {
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

	s.logger.Info("building search index...")
	if err := s.searchDB.BuildIndex(documents); err != nil {
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

func (s *Service) getDeletedFiles() ([]string, error) {
	allKeys, err := s.kvDB.GetAllKeys()
	if err != nil {
		s.logger.Error("failed to get all keys from database", "err", err.Error())
		return nil, fmt.Errorf("failed to get all keys from database: %w", err)
	}

	var deletedFiles []string
	for _, key := range allKeys {
		if key == kvdb.LastIndexTimeKey {
			continue
		}

		if _, err := os.Stat(key); os.IsNotExist(err) {
			deletedFiles = append(deletedFiles, key)
		}
	}

	return deletedFiles, nil
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
