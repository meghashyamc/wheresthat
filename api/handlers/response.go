package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type response struct {
	Data   any      `json:"data"`
	Errors []string `json:"errors"`
}

func writeResponse(c *gin.Context, data interface{}, statusCode int, errors []string) {

	if statusCode == http.StatusNoContent {
		c.JSON(statusCode, nil)
		return

	}

	response := response{
		Data:   data,
		Errors: errors,
	}

	c.JSON(statusCode, response)
}

type Pagination struct {
	CurrentPage  int  `json:"current_page"`
	PageSize     int  `json:"page_size"`
	TotalPages   int  `json:"total_pages"`
	HasNextPage  bool `json:"has_next_page"`
	HasPrevPage  bool `json:"has_prev_page"`
	TotalResults int  `json:"total_results"`
}

func calculatePagination(total, limit, offset int) Pagination {
	pageSize := limit
	currentPage := (offset / limit) + 1
	totalPages := (total + pageSize - 1) / pageSize

	if totalPages == 0 {
		totalPages = 1
	}

	return Pagination{
		CurrentPage:  currentPage,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		HasNextPage:  currentPage < totalPages,
		HasPrevPage:  currentPage > 1,
		TotalResults: total,
	}
}
