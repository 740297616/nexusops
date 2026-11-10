package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

func TestHealthAndNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter("test-version", func(context.Context) error { return nil })
	tests := []struct {
		path   string
		status int
		code   string
	}{
		{"/api/v1/health", http.StatusOK, ""},
		{"/api/v1/missing", http.StatusNotFound, "not_found"},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d", response.Code, tc.status)
			}
			var body struct {
				Data  map[string]any      `json:"data"`
				Error *platform.ErrorBody `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tc.code == "" {
				if body.Error != nil || body.Data["status"] != "ok" || body.Data["version"] != "test-version" {
					t.Fatalf("unexpected health response: %+v", body)
				}
			} else if body.Data != nil || body.Error == nil || body.Error.Code != tc.code {
				t.Fatalf("unexpected error response: %+v", body)
			}
		})
	}
}

func TestHealthAndReadinessWhenDatabaseUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter("test-version", func(context.Context) error { return errors.New("secret connection detail") })
	for _, tc := range []struct {
		path   string
		code   int
		status string
	}{{"/api/v1/health", 200, "ok"}, {"/api/v1/ready", 503, "unavailable"}} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		var body struct {
			Data struct {
				Status   string `json:"status"`
				Database struct {
					Status string `json:"status"`
				} `json:"database"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != tc.code || body.Data.Status != tc.status || body.Data.Database.Status != "unavailable" {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Fatal("response leaked connection detail")
		}
	}
}
