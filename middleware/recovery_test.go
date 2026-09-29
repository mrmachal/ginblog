package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecoveryKeepsAccessLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := gin.New()
	r.Use(Logger(), Recovery())
	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	out := buf.String()
	if !strings.Contains(out, "panic recovered") {
		t.Errorf("missing panic log: %s", out)
	}
	if !strings.Contains(out, `"msg":"request"`) {
		t.Errorf("access log lost after panic: %s", out)
	}
	if !strings.Contains(out, "panic: boom") {
		t.Errorf("panic not attached to access log: %s", out)
	}
	t.Log(out)
}
