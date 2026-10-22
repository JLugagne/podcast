// Package httpjson decodes and writes JSON HTTP bodies with a size limit and
// strict field checking.
package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrBadRequest marks a body that could not be decoded.
var ErrBadRequest = errors.New("httpjson: bad request body")

// ErrTooLarge marks a body that exceeded the configured limit.
var ErrTooLarge = errors.New("httpjson: request body too large")

// ErrorBody is the JSON shape of every API error.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Decode reads a JSON body into dst. It rejects unknown fields and bodies
// larger than maxBytes; the error wraps ErrBadRequest or ErrTooLarge so
// handlers can map it with errors.Is.
func Decode(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if maxBytes <= 0 {
		return fmt.Errorf("%w: max bytes must be positive", ErrBadRequest)
	}
	body := http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return fmt.Errorf("%w: limit %d bytes", ErrTooLarge, maxBytes)
		}
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: empty body", ErrBadRequest)
		}
		return fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	if dec.More() {
		return fmt.Errorf("%w: trailing data after JSON value", ErrBadRequest)
	}
	return nil
}

// Respond writes v as JSON with the given status.
func Respond(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// Error writes the {code, message} error body with the given status.
func Error(w http.ResponseWriter, _ *http.Request, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Code: code, Message: message})
}

// NoContent writes a 204 response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
