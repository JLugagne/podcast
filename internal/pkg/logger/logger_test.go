package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSONAtLevel(t *testing.T) {
	var buf bytes.Buffer
	l, err := New("json", "debug", &buf)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Debug("hello", slog.String("key", "value"))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log line %q: %v", buf.String(), err)
	}
	if entry["msg"] != "hello" || entry["key"] != "value" {
		t.Errorf("unexpected entry: %v", entry)
	}
	if entry["level"] != "DEBUG" {
		t.Errorf("level = %v, want DEBUG", entry["level"])
	}
}

func TestNewTextFiltersBelowLevel(t *testing.T) {
	var buf bytes.Buffer
	l, err := New("text", "info", &buf)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Debug("hidden")
	if buf.Len() != 0 {
		t.Errorf("debug log leaked through info level: %q", buf.String())
	}
	l.Info("shown")
	if !strings.Contains(buf.String(), "shown") {
		t.Errorf("info log missing: %q", buf.String())
	}
}

func TestNewUnknownFormatFallsBackToText(t *testing.T) {
	var buf bytes.Buffer
	l, err := New("bogus", "", &buf)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Info("hello")
	if !strings.Contains(buf.String(), "level=INFO") {
		t.Errorf("expected text handler output, got %q", buf.String())
	}
}

func TestNewInvalidLevel(t *testing.T) {
	if _, err := New("text", "loud", nil); err == nil {
		t.Fatal("New with invalid level = nil error, want error")
	}
}

func TestWithLoggerRoundTrip(t *testing.T) {
	l, err := New("text", "info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := WithLogger(context.Background(), l)
	if got := FromContext(ctx); got != l {
		t.Errorf("FromContext = %p, want %p", got, l)
	}
}

func TestFromContextDefaults(t *testing.T) {
	if got := FromContext(context.Background()); got != slog.Default() {
		t.Errorf("FromContext(empty) = %p, want slog.Default()", got)
	}
	ctx := WithLogger(context.Background(), nil)
	if got := FromContext(ctx); got != slog.Default() {
		t.Errorf("FromContext(nil logger) = %p, want slog.Default()", got)
	}
}
