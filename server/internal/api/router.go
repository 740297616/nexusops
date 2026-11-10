package api

import (
	"net/http"

	"github.com/740297616/nexusops/server/internal/middleware"
	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

func NewRouter(version string) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestLogger(), middleware.Recovery())
	router.GET("/api/v1/health", func(c *gin.Context) {
		platform.Respond(c, http.StatusOK, gin.H{"status": "ok", "version": version}, nil)
	})
	router.NoRoute(func(c *gin.Context) {
		platform.Respond(c, http.StatusNotFound, nil, &platform.ErrorBody{Code: "not_found", Message: "Resource not found"})
	})
	return router
}
