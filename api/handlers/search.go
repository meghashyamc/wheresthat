package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/db"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/services/search"
	"github.com/meghashyamc/wheresthat/validation"
)

const defaultSearchLimit = 20

type SearchRequest struct {
	Query  string `form:"query" validate:"required,valid_query,min=1,max=1000"`
	Limit  int    `form:"limit" validate:"min=1,max=100"`
	Offset int    `form:"offset" validate:"min=0"`
}

func (r *SearchRequest) setDefaults() {
	if r.Limit == 0 {
		r.Limit = defaultSearchLimit
	}
}

func SetupSearch(router *gin.Engine, logger logger.Logger, db db.DB, validator *validation.Validator) {
	service := search.New(logger, db)
	router.GET("/search", handleSearch(service, logger, validator))

}

func handleSearch(service *search.Service, logger logger.Logger, validator *validation.Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := SearchRequest{}
		if err := c.ShouldBindQuery(&request); err != nil {
			logger.Warn("could not extract expected params from search request", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusUnprocessableEntity, []string{"failed to extract request body parameters"})
			return
		}
		request.setDefaults()

		if err := validator.Validate(request); err != nil {
			logger.Warn("could not validate search request", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusNotAcceptable, []string{err.Error()})
			return
		}

		results, err := service.Search(request.Query, request.Limit, request.Offset)
		if err != nil {
			logger.Error("search failed", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusInternalServerError, []string{err.Error()})
			return
		}

		writeResponse(c, results, http.StatusOK, nil)
	}
}
