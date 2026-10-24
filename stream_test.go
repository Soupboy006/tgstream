package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

type fakeAPI struct {
	data         []byte
	calls        int
	expire       int
	alwaysExpire bool
	err          error
}

func (f *fakeAPI) UploadGetFile(_ context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if f.alwaysExpire || f.expire > 0 {
		if f.expire > 0 {
			f.expire--
		}
		return nil, tgerr.New(400, "FILE_REFERENCE_EXPIRED")
	}
	if req.Offset >= int64(len(f.data)) {
		return &tg.UploadFile{}, nil
	}
	end := req.Offset + int64(req.Limit)
	if end > int64(len(f.data)) {
		end = int64(len(f.data))
	}
	return &tg.UploadFile{Bytes: f.data[req.Offset:end]}, nil
}

func testDoc(data []byte) *tg.Document {
	return &tg.Document{
		ID:            1,
		AccessHash:    2,
		FileReference: []byte("ref"),
		Size:          int64(len(data)),
		MimeType:      "video/mp4",
	}
}

func newTestHandler(api fileAPI, refresh refreshFunc, data []byte) http.Handler {
	return newStreamHandler(api, refresh, testDoc(data), "test-token")
}

func noRefresh(context.Context) (*tg.Document, error) {
	return nil, errors.New("no refresh")
}

func doRequest(t *testing.T, h http.Handler, method, path, rangeHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func body(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	b, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestStreamFullFile(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "GET", "/play/test-token", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := body(t, rec); got != "0123456789" {
		t.Fatalf("body = %q", got)
	}
	if got := rec.Header().Get("Content-Length"); got != "10" {
		t.Fatalf("Content-Length = %q", got)
	}
	if got := rec.Header().Get("Content-Range"); got != "" {
		t.Fatalf("Content-Range on 200 = %q, want empty", got)
	}
	if got := rec.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("Accept-Ranges = %q", got)
	}
}

func TestStreamRange(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "GET", "/play/test-token", "bytes=2-5")

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", rec.Code)
	}
	if got := body(t, rec); got != "2345" {
		t.Fatalf("body = %q, want 2345", got)
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("Content-Range = %q", got)
	}
}

func TestStreamSuffixRange(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "GET", "/play/test-token", "bytes=-3")

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", rec.Code)
	}
	if got := body(t, rec); got != "789" {
		t.Fatalf("body = %q, want 789", got)
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 7-9/10" {
		t.Fatalf("Content-Range = %q", got)
	}
}

func TestStreamUnsatisfiableRange(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "GET", "/play/test-token", "bytes=10-")

	if rec.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("status = %d, want 416", rec.Code)
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes */10" {
		t.Fatalf("Content-Range = %q", got)
	}
}

func TestStreamInvalidRange(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "GET", "/play/test-token", "items=0-1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestStreamHeadDoesNotDownload(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "HEAD", "/play/test-token", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := body(t, rec); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
	if got := rec.Header().Get("Content-Length"); got != "10" {
		t.Fatalf("Content-Length = %q", got)
	}
	if api.calls != 0 {
		t.Fatalf("HEAD made %d upstream calls, want 0", api.calls)
	}
}

func TestStreamHeadRange(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "HEAD", "/play/test-token", "bytes=2-5")

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", rec.Code)
	}
	if got := rec.Header().Get("Content-Length"); got != "4" {
		t.Fatalf("Content-Length = %q", got)
	}
	if api.calls != 0 {
		t.Fatalf("HEAD made %d upstream calls, want 0", api.calls)
	}
}

func TestStreamBadToken(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "GET", "/play/wrong", "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if api.calls != 0 {
		t.Fatalf("bad token made %d upstream calls, want 0", api.calls)
	}
}

func TestStreamRefreshFileReference(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789"), expire: 1}
	refreshes := 0
	refresh := func(context.Context) (*tg.Document, error) {
		refreshes++
		return testDoc(api.data), nil
	}
	rec := doRequest(t, newTestHandler(api, refresh, api.data), "GET", "/play/test-token", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := body(t, rec); got != "0123456789" {
		t.Fatalf("body = %q", got)
	}
	if refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", refreshes)
	}
}

func TestStreamRefreshGivesUp(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789"), alwaysExpire: true}
	refreshes := 0
	refresh := func(context.Context) (*tg.Document, error) {
		refreshes++
		return testDoc(api.data), nil
	}
	rec := doRequest(t, newTestHandler(api, refresh, api.data), "GET", "/play/test-token", "")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if refreshes > 2 {
		t.Fatalf("refreshes = %d, want at most 2", refreshes)
	}
}

func TestStreamUpstreamError(t *testing.T) {
	api := &fakeAPI{data: []byte("0123456789"), err: errors.New("boom")}
	rec := doRequest(t, newTestHandler(api, noRefresh, api.data), "GET", "/play/test-token", "")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}
