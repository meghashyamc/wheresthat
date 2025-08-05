package api

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	ginpprof "github.com/gin-contrib/pprof"
	"github.com/meghashyamc/wheresthat/api/handlers"
	"github.com/meghashyamc/wheresthat/ui"
)

func (s *server) setupRoutes(ctx context.Context, router *gin.Engine) {
	router.GET("/health", health())

	// Setup pprof endpoints for profiling
	ginpprof.Register(router)

	// Serve static UI files
	router.StaticFS("/ui", http.FS(ui.Files))
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/index.html")
	})

	handlers.SetupIndex(ctx, router, s.logger, s.indexer, s.metadataStore, s.validator)
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
