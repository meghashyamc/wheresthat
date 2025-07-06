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
