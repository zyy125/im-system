package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/middleware"
)

func TestRespondErrorLogsHandlerCallsite(t *testing.T) {
	buffer, restore := installHandlerTestLogger(t)
	defer restore()

	engine := gin.New()
	engine.Use(middleware.RequestLogger())
	engine.Use(middleware.Recovery())
	engine.GET("/respond", func(c *gin.Context) {
		respondError(c, apperr.InvalidArgument("bad request"))
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/respond", nil)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	entry := decodeSingleHandlerLogEntry(t, buffer)
	file, _ := entry["file"].(string)
	if !strings.HasSuffix(file, "internal/handler/helpers_test.go") {
		t.Fatalf("expected file to point to handler callsite, got %v", entry["file"])
	}
	function, _ := entry["function"].(string)
	if !strings.Contains(function, "TestRespondErrorLogsHandlerCallsite") {
		t.Fatalf("expected test handler function, got %v", entry["function"])
	}
}

func TestBindJSONLogsHandlerCallsite(t *testing.T) {
	buffer, restore := installHandlerTestLogger(t)
	defer restore()

	type requestBody struct {
		Name string `json:"name"`
	}

	engine := gin.New()
	engine.Use(middleware.RequestLogger())
	engine.Use(middleware.Recovery())
	engine.POST("/bind", func(c *gin.Context) {
		var body requestBody
		if !bindJSON(c, &body) {
			return
		}
		respondOK(c, gin.H{"name": body.Name})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	entry := decodeSingleHandlerLogEntry(t, buffer)
	file, _ := entry["file"].(string)
	if !strings.HasSuffix(file, "internal/handler/helpers_test.go") {
		t.Fatalf("expected file to point to bindJSON caller, got %v", entry["file"])
	}
	function, _ := entry["function"].(string)
	if !strings.Contains(function, "TestBindJSONLogsHandlerCallsite") {
		t.Fatalf("expected bindJSON caller function, got %v", entry["function"])
	}
}

func installHandlerTestLogger(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	buffer := &bytes.Buffer{}
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	previousLogger := slog.Default()
	logger := slog.New(slog.NewJSONHandler(buffer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	return buffer, func() {
		slog.SetDefault(previousLogger)
		gin.SetMode(previousMode)
	}
}

func decodeSingleHandlerLogEntry(t *testing.T, buffer *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one log line, got %d: %q", len(lines), buffer.String())
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}
	return entry
}
