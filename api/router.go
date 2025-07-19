package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/api/handlers"
	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/ui"
	"github.com/meghashyamc/wheresthat/validation"
)

func setupRoutes(ctx context.Context, router *gin.Engine, logger logger.Logger, searchDB searchdb.DB, kvDB kvdb.DB, validator *validation.Validator) {
	router.GET("/health", health())

	// Serve static UI files
	router.StaticFS("/ui", http.FS(ui.Files))
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/index.html")
	})

	handlers.SetupIndex(ctx, router, logger, searchDB, kvDB, validator)
	handlers.SetupSearch(router, logger, searchDB, validator)

}

func health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	}
}

func newRouter() *gin.Engine {
	router := gin.Default()
	router.UseRawPath = true
	router.Use(_CORSMiddleware())
	router.Use(gin.Recovery())
	router.Use(authMiddleware())

	return router
}
