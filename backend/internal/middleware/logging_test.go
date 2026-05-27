package middleware

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
	"github.com/zyy125/im-system/pkg/response"
)

func TestRequestLoggerInfoForSuccess(t *testing.T) {
	restore := installTestLogger(t)
	defer restore()

	engine := gin.New()
	engine.Use(RequestLogger())
	engine.Use(Recovery())
	engine.GET("/ok", func(c *gin.Context) {
		response.Success(c, http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	entry := readSingleLogEntry(t)
	if entry["level"] != "INFO" {
		t.Fatalf("expected INFO level, got %v", entry["level"])
	}
	if entry["msg"] != "http request completed" {
		t.Fatalf("expected request log message, got %v", entry["msg"])
	}
	if entry["event_type"] != "http_request" {
		t.Fatalf("expected event_type http_request, got %v", entry["event_type"])
	}
	if entry["method"] != http.MethodGet {
		t.Fatalf("expected method GET, got %v", entry["method"])
	}
	if entry["path"] != "/ok" {
		t.Fatalf("expected path /ok, got %v", entry["path"])
	}
	if status, ok := entry["status"].(float64); !ok || int(status) != http.StatusOK {
		t.Fatalf("expected status 200 in log, got %v", entry["status"])
	}
}

func TestRequestLoggerWarnUsesRespondErrorCallsite(t *testing.T) {
	restore := installTestLogger(t)
	defer restore()

	engine := gin.New()
	engine.Use(RequestLogger())
	engine.Use(Recovery())
	engine.GET("/warn", func(c *gin.Context) {
		response.FailError(c, apperr.InvalidArgument("bad request"))
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/warn", nil)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	entry := readSingleLogEntry(t)
	if entry["level"] != "WARN" {
		t.Fatalf("expected WARN level, got %v", entry["level"])
	}
	if entry["error_code"] != string(apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid argument code, got %v", entry["error_code"])
	}
	if entry["error_message"] != "bad request" {
		t.Fatalf("expected error message, got %v", entry["error_message"])
	}
	file, _ := entry["file"].(string)
	if !strings.HasSuffix(file, "internal/middleware/logging_test.go") {
		t.Fatalf("expected file to point to test handler, got %v", entry["file"])
	}
	function, _ := entry["function"].(string)
	if !strings.Contains(function, "TestRequestLoggerWarnUsesRespondErrorCallsite") {
		t.Fatalf("expected handler function in log, got %v", entry["function"])
	}
}

func TestRequestLoggerWarnUsesHelperCallsite(t *testing.T) {
	t.Skip("helper callsite coverage lives in internal/handler tests")
}

func TestRecoveryReturnsUnifiedErrorAndLogsOnce(t *testing.T) {
	restore := installTestLogger(t)
	defer restore()

	engine := gin.New()
	engine.Use(RequestLogger())
	engine.Use(Recovery())
	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", recorder.Code)
	}

	var body response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != string(apperr.CodeInternal) {
		t.Fatalf("expected internal code, got %s", body.Code)
	}
	if body.Message != "internal server error" {
		t.Fatalf("expected internal error message, got %s", body.Message)
	}

	entry := readSingleLogEntry(t)
	if entry["level"] != "ERROR" {
		t.Fatalf("expected ERROR level, got %v", entry["level"])
	}
	if entry["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("expected status 500 in log, got %v", entry["status"])
	}
	file, _ := entry["file"].(string)
	if !strings.HasSuffix(file, "internal/middleware/logging_test.go") {
		t.Fatalf("expected panic callsite file, got %v", entry["file"])
	}
	function, _ := entry["function"].(string)
	if !strings.Contains(function, "TestRecoveryReturnsUnifiedErrorAndLogsOnce") {
		t.Fatalf("expected panic handler function, got %v", entry["function"])
	}
}

var testLogBuffer bytes.Buffer

func installTestLogger(t *testing.T) func() {
	t.Helper()
	testLogBuffer.Reset()
	previous := gin.Mode()
	gin.SetMode(gin.TestMode)
	previousLogger := slog.Default()
	logger := slog.New(slog.NewJSONHandler(&testLogBuffer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	return func() {
		slog.SetDefault(previousLogger)
		gin.SetMode(previous)
	}
}

func readSingleLogEntry(t *testing.T) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(testLogBuffer.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one log line, got %d: %q", len(lines), testLogBuffer.String())
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}
	return entry
}
