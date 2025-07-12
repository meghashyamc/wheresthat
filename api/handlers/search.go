package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/services/search"
	"github.com/meghashyamc/wheresthat/validation"
)

const defaultResultsPerPage = 20

type SearchRequest struct {
	Query   string `form:"query" validate:"required,valid_query,min=1,max=1000"`
	PerPage int    `form:"per_page" validate:"min=0,max=100"`
	Page    int    `form:"page" validate:"min=0"`
}

func (r *SearchRequest) setDefaults() {
	if r.PerPage == 0 {
		r.PerPage = defaultResultsPerPage
	}

	if r.Page == 0 {
		r.Page = 1
	}
}

type SearchResponse struct {
	Results     []searchdb.Result `json:"results"`
	PageDetails Pagination        `json:"page_details"`
}

func SetupSearch(router *gin.Engine, logger logger.Logger, searchdb searchdb.DB, validator *validation.Validator) {
	service := search.New(logger, searchdb)
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

		limit := request.PerPage
		offset := (request.Page - 1) * request.PerPage
		results, err := service.Search(request.Query, limit, offset)
		if err != nil {
			logger.Error("search failed", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusInternalServerError, []string{err.Error()})
			return
		}

		searchResponse := SearchResponse{
			Results: results.Results,
			PageDetails: calculatePagination(
				int(results.Total),
				limit,
				offset),
		}

		writeResponse(c, searchResponse, http.StatusOK, nil)
	}
}
