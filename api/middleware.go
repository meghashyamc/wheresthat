package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/logger"
)

const HeaderPaginationTotalCount = "X-Pagination-Total-Count"

func loggingMiddleware(logger logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info("request", "method", c.Request.Method, "path", c.Request.URL.Path)
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO
		c.Next()
	}
}

// _CORSMiddleware starts with _ so that it is not imported outside of the server package.
func _CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Authentication, accept, origin, Cache-Control, X-Requested-With") // nolint:lll
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Expose-Headers", HeaderPaginationTotalCount)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)

			return
		}

		c.Next()
	}
}
