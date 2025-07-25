package search

import (
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
)

// Searcher represents the search database operations needed for search functionality
type Searcher interface {
	Search(queryString string, limit int, offset int) (*searchdb.Response, error)
}

type Service struct {
	logger   logger.Logger
	searcher Searcher
}

func New(logger logger.Logger, searcher Searcher) *Service {
	return &Service{
		logger:   logger,
		searcher: searcher,
	}
}

func (s *Service) Search(query string, limit int, offset int) (*searchdb.Response, error) {
	s.logger.Info("performing search", "query", query, "limit", limit, "offset", offset)

	// Perform search
	results, err := s.searcher.Search(query, limit, offset)
	if err != nil {
		s.logger.Error("search failed", "err", err.Error())
		return nil, err
	}

	s.logger.Info("search completed", "total_results", results.Total, "returned_results", len(results.Results))

	return results, nil
}
