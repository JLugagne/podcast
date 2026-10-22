package commands

import (
	"bufio"
	"errors"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
)

// multipartOverheadBytes is the slack allowed on top of the configured file
// size limit for boundaries and part headers.
const multipartOverheadBytes = 1 << 20

// maxDurationFieldBytes bounds the duration_seconds multipart field.
const maxDurationFieldBytes = 64

// upload is one streamed multipart file part.
type upload struct {
	Reader      io.Reader
	ContentType string
	FileName    string
	part        *multipart.Part
}

// close releases the underlying part; the body stream must not be used after.
func (u upload) close() {
	_ = u.part.Close()
}

// readUpload walks the multipart body and returns the part named "file".
//
// Every field part MUST come before the file part: the file is streamed
// straight to the use case, so the parts that follow it are never read. The
// optional duration_seconds field is parsed when withDuration is set.
func readUpload(w http.ResponseWriter, r *http.Request, maxFileBytes int64, withDuration bool) (upload, float64, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileBytes+multipartOverheadBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		return upload{}, 0, shared.MultipartError(err)
	}

	var duration float64
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return upload{}, 0, shared.ErrMissingFile
		}
		if err != nil {
			return upload{}, 0, shared.MultipartError(err)
		}

		switch part.FormName() {
		case "file":
			return newUpload(part, maxFileBytes), duration, nil
		case "duration_seconds":
			if !withDuration {
				_ = part.Close()
				continue
			}
			raw, err := io.ReadAll(io.LimitReader(part, maxDurationFieldBytes))
			_ = part.Close()
			if err != nil {
				return upload{}, 0, shared.MultipartError(err)
			}
			duration, err = parseDuration(string(raw))
			if err != nil {
				return upload{}, 0, err
			}
		default:
			_ = part.Close()
		}
	}
}

// newUpload wraps a file part. When the part carries no usable content type,
// the first bytes are sniffed so the client declaration is not trusted blindly.
func newUpload(part *multipart.Part, maxFileBytes int64) upload {
	file := upload{
		Reader:      &capReader{r: part, left: maxFileBytes, limit: maxFileBytes},
		ContentType: part.Header.Get("Content-Type"),
		FileName:    part.FileName(),
		part:        part,
	}
	switch domain.NormalizeContentType(file.ContentType) {
	case "", "application/octet-stream":
		buffered := bufio.NewReader(file.Reader)
		head, _ := buffered.Peek(512)
		file.Reader = buffered
		file.ContentType = http.DetectContentType(head)
	}
	return file
}

// capReader fails with *http.MaxBytesError once more than limit bytes have
// been read, so one file part cannot exceed its configured limit even when the
// multipart envelope still has slack.
type capReader struct {
	r     io.Reader
	left  int64
	limit int64
}

func (c *capReader) Read(p []byte) (int, error) {
	if c.left <= 0 {
		var probe [1]byte
		n, err := c.r.Read(probe[:])
		if n > 0 {
			return 0, &http.MaxBytesError{Limit: c.limit}
		}
		return 0, err
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	return n, err
}

// parseDuration parses the duration_seconds field. A zero duration means
// "unknown" and is accepted.
func parseDuration(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, shared.ErrInvalidDuration
	}
	return value, nil
}
