package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/db"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/services/search"
	"github.com/meghashyamc/wheresthat/validation"
)

func SetupSearch(router *gin.Engine, logger logger.Logger, db db.DB, validator *validation.Validator) {
	service := search.New(logger, db)
	router.POST("/search", handleSearch(service, validator))

}

func handleSearch(service *search.Service, validator *validation.Validator) gin.HandlerFunc {
	return func(c *gin.Context) {

		c.String(http.StatusOK, "OK")
	}
}
