package httpjson

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type payload struct {
	Name string `json:"name"`
}

func TestDecodeValid(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(`{"name":"box"}`))

	var got payload
	if err := Decode(rec, req, &got, 1<<20); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Name != "box" {
		t.Errorf("Name = %q, want box", got.Name)
	}
}

func TestDecodeErrors(t *testing.T) {
	tests := map[string]struct {
		body    string
		max     int64
		wantErr error
	}{
		"unknown field": {body: `{"name":"box","extra":1}`, max: 1 << 20, wantErr: ErrBadRequest},
		"malformed":     {body: `{"name":`, max: 1 << 20, wantErr: ErrBadRequest},
		"empty":         {body: ``, max: 1 << 20, wantErr: ErrBadRequest},
		"trailing data": {body: `{"name":"a"}{"name":"b"}`, max: 1 << 20, wantErr: ErrBadRequest},
		"too large":     {body: `{"name":"` + strings.Repeat("a", 64) + `"}`, max: 8, wantErr: ErrTooLarge},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(tc.body))
			var got payload
			err := Decode(rec, req, &got, tc.max)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Decode error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestRespond(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := Respond(rec, http.StatusCreated, payload{Name: "box"}); err != nil {
		t.Fatalf("Respond: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	var got payload
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Name != "box" {
		t.Errorf("Name = %q, want box", got.Name)
	}
}

func TestError(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil), http.StatusUnprocessableEntity, "INVALID", "bad input")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	var got ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Code != "INVALID" || got.Message != "bad input" {
		t.Errorf("body = %+v", got)
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	NoContent(rec)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}
