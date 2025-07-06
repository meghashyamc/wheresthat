package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/db"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/services/index"
	"github.com/meghashyamc/wheresthat/validation"
)

type IndexRequest struct {
	Path string `json:"path" validate:"valid_path"`
}

func SetupIndex(router *gin.Engine, logger logger.Logger, db db.DB, validator *validation.Validator) {
	service := index.New(logger, db)
	router.POST("/index", handleIndex(service, logger, validator))

}

func handleIndex(index *index.Service, logger logger.Logger, validator *validation.Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := IndexRequest{}
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.Warn("could not extract expected query params from the input for get notifications", "err", err.Error())
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

		if err := index.Create(request.Path); err != nil {
			logger.Warn("could not create index", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusInternalServerError, []string{err.Error()})
			return
		}

		writeResponse(c, nil, http.StatusNoContent, nil)
	}
}
