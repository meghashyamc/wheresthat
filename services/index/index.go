package index

import (
	"log"
	"log/slog"
	"os"

	"github.com/meghashyamc/wheresthat/db"
	"github.com/meghashyamc/wheresthat/logger"
)

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
	var documents []db.SearchDocument
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
	if err := s.db.BuildIndex(documents, indexPath); err != nil {
		log.Fatal(err)
	}

	s.logger.Info("Index built successfully!")

	indexStat, _ := os.Stat(indexPath)
	s.logger.Info("Index size in bytes", slog.Int64("size", indexStat.Size()))

	return nil
}
