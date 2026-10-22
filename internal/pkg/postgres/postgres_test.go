package postgres

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewPoolInvalidDSN(t *testing.T) {
	if _, err := NewPool(context.Background(), "postgres://user:pass@host:notaport/db"); err == nil {
		t.Fatal("NewPool with invalid dsn = nil error, want error")
	}
}

func TestNewPoolGivesUpWhenContextExpires(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := NewPool(ctx, "postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatal("NewPool to a closed port = nil error, want error")
	}
	if !strings.Contains(err.Error(), "postgres: connect") {
		t.Errorf("error = %v, want a connect error", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("NewPool took %s, expected it to stop at the context deadline", elapsed)
	}
}
