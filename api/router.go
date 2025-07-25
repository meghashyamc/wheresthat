package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/api/handlers"
	"github.com/meghashyamc/wheresthat/ui"
)

func (s *server) setupRoutes(ctx context.Context, router *gin.Engine) {
	router.GET("/health", health())

	// Serve static UI files
	router.StaticFS("/ui", http.FS(ui.Files))
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/index.html")
	})

	handlers.SetupIndex(ctx, router, s.logger, s.indexer, s.kvDB, s.validator)
	handlers.SetupSearch(router, s.logger, s.searcher, s.validator)

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
