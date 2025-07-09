package index

import (
	"log"
	"log/slog"

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
	s.logger.Info("discovering files...")
	files, err := discoverFiles(rootPath)
	if err != nil {
		s.logger.Error("could not discover files", "err", err.Error())
		return err
	}
	s.logger.Info("discovered files", slog.Int("num_of_files", len(files)))

	// Step 2: Extract content and process
	s.logger.Info("Processing files...")
	var documents []searchdb.Document
	for i, file := range files {
		if i%100 == 0 {
			s.logger.Info("Processed %d/%d files\n", i, len(files))
		}

		doc, err := extractContent(file)
		if err != nil {
			s.logger.Error("Error processing %s: %v\n", file.Path, err)
			continue
		}

		documents = append(documents, *doc)
	}

	s.logger.Info("Building search index...")
	if err := s.searchdb.BuildIndex(documents); err != nil {
		log.Fatal(err)
	}

	s.logger.Info("Index built successfully!")

	return nil
}
