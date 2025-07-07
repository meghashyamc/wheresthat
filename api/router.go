package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/api/handlers"
	"github.com/meghashyamc/wheresthat/db"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/validation"
)

func newRouter() *gin.Engine {

	router := setupRouter()
	logger := logger.New()
	router.Use(loggingMiddleware(logger))

	db := db.New(logger)
	validator, err := validation.New(logger)
	if err != nil {
		logger.Error("error creating validator", "err", err.Error())
		os.Exit(1)
	}

	setupRoutes(router, logger, db, validator)

	return router
}
func setupRoutes(router *gin.Engine, logger logger.Logger, db db.DB, validator *validation.Validator) {
	router.GET("/health", health())

	handlers.SetupIndex(router, logger, db, validator)
	handlers.SetupSearch(router, logger, db, validator)
}

func health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	}
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.UseRawPath = true
	router.Use(_CORSMiddleware())
	router.Use(gin.Recovery())
	router.Use(authMiddleware())

	return router
}
