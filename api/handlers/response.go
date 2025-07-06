package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type response struct {
	Data   any
	Errors []string
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
