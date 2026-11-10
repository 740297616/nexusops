package api

import (
	"context"
	"net/http"

	"github.com/740297616/nexusops/server/internal/middleware"
	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

func NewRouter(version string, ping func(context.Context) error) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestLogger(), middleware.Recovery())
	router.GET("/api/v1/health", func(c *gin.Context) {
		dbStatus := "ok"
		if ping(c.Request.Context()) != nil {
			dbStatus = "unavailable"
		}
		platform.Respond(c, http.StatusOK, gin.H{"status": "ok", "version": version, "database": gin.H{"status": dbStatus}}, nil)
	})
	router.GET("/api/v1/ready", func(c *gin.Context) {
		if ping(c.Request.Context()) != nil {
			platform.Respond(c, http.StatusServiceUnavailable, gin.H{"status": "unavailable", "database": gin.H{"status": "unavailable"}}, nil)
			return
		}
		platform.Respond(c, http.StatusOK, gin.H{"status": "ok", "database": gin.H{"status": "ok"}}, nil)
	})
	router.NoRoute(func(c *gin.Context) {
		platform.Respond(c, http.StatusNotFound, nil, &platform.ErrorBody{Code: "not_found", Message: "Resource not found"})
	})
	return router
}
