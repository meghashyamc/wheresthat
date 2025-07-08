package search

import (
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

func (s *Service) Search(query string, limit int, offset int) (*db.SearchResponse, error) {
	s.logger.Info("performing search", "query", query, "limit", limit, "offset", offset)

	// Perform search
	results, err := s.db.Search(query, limit, offset)
	if err != nil {
		s.logger.Error("search failed", "err", err.Error())
		return nil, err
	}

	s.logger.Info("search completed", "total_results", results.Total, "returned_results", len(results.Results))

	return results, nil
}
