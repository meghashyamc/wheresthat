package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/services/index"
	"github.com/meghashyamc/wheresthat/validation"
)

type IndexRequest struct {
	Path string `json:"path" validate:"required,valid_path"`
}

func SetupIndex(ctx context.Context, router *gin.Engine, logger logger.Logger, searchDB searchdb.DB, kvDB kvdb.DB, validator *validation.Validator) {
	service := index.New(ctx, logger, searchDB, kvDB)
	router.POST("/index", handleCreateIndex(service, logger, validator))
}

func handleCreateIndex(index *index.Service, logger logger.Logger, validator *validation.Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := IndexRequest{}
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.Warn("could not extract expected query params from the input", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusUnprocessableEntity, []string{"failed to extract request body parameters"})
			return
		}

		if err := validator.Validate(request); err != nil {
			logger.Warn("could not validate request", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusNotAcceptable, []string{err.Error()})
			return
		}

		index.Create(request.Path)

		writeResponse(c, nil, http.StatusNoContent, nil)
	}
}
