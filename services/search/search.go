package search

import (
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
)

type Service struct {
	logger   logger.Logger
	searchDB searchdb.DB
}

func New(logger logger.Logger, searchDB searchdb.DB) *Service {
	return &Service{
		logger:   logger,
		searchDB: searchDB,
	}

}

func (s *Service) Search(query string, limit int, offset int) (*searchdb.Response, error) {
	s.logger.Info("performing search", "query", query, "limit", limit, "offset", offset)

	// Perform search
	results, err := s.searchDB.Search(query, limit, offset)
	if err != nil {
		s.logger.Error("search failed", "err", err.Error())
		return nil, err
	}

	s.logger.Info("search completed", "total_results", results.Total, "returned_results", len(results.Results))

	return results, nil
}
