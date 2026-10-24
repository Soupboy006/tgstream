package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

const chunkSize = 1024 * 1024

// fileAPI is the subset of *tg.Client needed to serve a stream.
type fileAPI interface {
	UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error)
}

// refreshFunc re-fetches the message to obtain a fresh file reference.
type refreshFunc func(context.Context) (*tg.Document, error)

type streamer struct {
	api        fileAPI
	refresh    refreshFunc
	docID      int64
	accessHash int64
	size       int64
	mime       string

	mu      sync.RWMutex
	fileRef []byte
}

func newStreamHandler(api fileAPI, refresh refreshFunc, doc *tg.Document, token string) http.Handler {
	s := &streamer{
		api:        api,
		refresh:    refresh,
		docID:      doc.ID,
		accessHash: doc.AccessHash,
		size:       doc.Size,
		mime:       doc.MimeType,
		fileRef:    bytes.Clone(doc.FileReference),
	}
	if s.mime == "" {
		s.mime = "application/octet-stream"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/play/"+token {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.serve(w, r)
	})
}

func (s *streamer) serve(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseRange(r.Header.Get("Range"), s.size)
	if err != nil {
		if errors.Is(err, errRangeNotSatisfiable) {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", s.size))
			http.Error(w, "range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		http.Error(w, "invalid range", http.StatusBadRequest)
		return
	}

	status := http.StatusOK
	if r.Header.Get("Range") != "" {
		status = http.StatusPartialContent
	}

	w.Header().Set("Content-Type", s.mime)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	if status == http.StatusPartialContent {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, s.size))
	}

	if r.Method == http.MethodHead || end < start {
		w.WriteHeader(status)
		return
	}

	offset := start - start%chunkSize
	wroteHeader := false
	for offset <= end {
		data, err := s.fetch(r.Context(), offset)
		if err != nil {
			if !wroteHeader {
				http.Error(w, "failed to fetch file", http.StatusBadGateway)
				return
			}
			log.Println("stream error:", err)
			return
		}
		if len(data) == 0 {
			if !wroteHeader {
				http.Error(w, "empty file chunk", http.StatusBadGateway)
			}
			return
		}
		if !wroteHeader {
			w.WriteHeader(status)
			wroteHeader = true
		}

		chunkStart := start - offset
		if chunkStart < 0 {
			chunkStart = 0
		}
		chunkEnd := int64(len(data))
		if offset+chunkEnd > end+1 {
			chunkEnd = end + 1 - offset
		}
		if chunkStart < chunkEnd {
			if _, err := w.Write(data[chunkStart:chunkEnd]); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}

		offset += int64(len(data))
	}
}

// fetch returns up to chunkSize bytes at offset, refreshing the file
// reference when Telegram says it expired.
func (s *streamer) fetch(ctx context.Context, offset int64) ([]byte, error) {
	for attempt := 0; attempt < 3; attempt++ {
		s.mu.RLock()
		ref := bytes.Clone(s.fileRef)
		s.mu.RUnlock()

		result, err := s.api.UploadGetFile(ctx, &tg.UploadGetFileRequest{
			Location: &tg.InputDocumentFileLocation{
				ID:            s.docID,
				AccessHash:    s.accessHash,
				FileReference: ref,
			},
			Offset: offset,
			Limit:  chunkSize,
		})
		if err == nil {
			file, ok := result.(*tg.UploadFile)
			if !ok {
				return nil, fmt.Errorf("unexpected result type: %T", result)
			}
			return file.Bytes, nil
		}
		if !tgerr.Is(err, "FILE_REFERENCE_EXPIRED") && !tgerr.Is(err, "FILE_REFERENCE_INVALID") {
			return nil, err
		}
		if attempt == 2 {
			break
		}

		s.mu.Lock()
		if bytes.Equal(s.fileRef, ref) {
			newDoc, rerr := s.refresh(ctx)
			if rerr != nil {
				s.mu.Unlock()
				return nil, fmt.Errorf("refresh file reference: %w", rerr)
			}
			s.fileRef = bytes.Clone(newDoc.FileReference)
		}
		s.mu.Unlock()
	}

	return nil, errors.New("file reference kept expiring")
}

func startStreamServer(api fileAPI, refresh refreshFunc, doc *tg.Document) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen: %w", err)
	}

	srv := &http.Server{
		Handler:           newStreamHandler(api, refresh, doc, token),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("server error:", err)
		}
	}()

	return "http://" + ln.Addr().String() + "/play/" + token, nil
}

func randomToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
