package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("method=%s route=%s status=%d duration=%s", c.Request.Method, c.FullPath(), c.Writer.Status(), time.Since(start))
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				log.Print("request panic")
				c.Abort()
				if !c.Writer.Written() {
					platform.Respond(c, http.StatusInternalServerError, nil, &platform.ErrorBody{Code: "internal_error", Message: "Internal server error"})
				}
			}
		}()
		c.Next()
	}
}
