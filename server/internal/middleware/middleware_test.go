package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(*gin.Context) { panic("sensitive detail") })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
	var body platform.Response
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError || body.Data != nil || body.Error == nil || body.Error.Code != "internal_error" || body.Error.Message != "Internal server error" {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}
