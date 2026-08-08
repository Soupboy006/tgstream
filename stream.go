package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"
)

func startStreamServer(ctx context.Context, api *tg.Client, channelID, accessHash int64, doc *tg.Document) string {
	loc := &tg.InputDocumentFileLocation{
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}

	mime := doc.MimeType
	if mime == "" {
		mime = "video/x-matroska"
	}
	fileSize := doc.Size

	mux := http.NewServeMux()
	mux.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		start, end := int64(0), fileSize-1
		status := http.StatusOK
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			status = http.StatusPartialContent
			rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
			bounds := strings.Split(rangeHeader, "-")
			if bounds[0] != "" {
				start, _ = strconv.ParseInt(bounds[0], 10, 64)
			}
			if len(bounds) > 1 && bounds[1] != "" {
				end, _ = strconv.ParseInt(bounds[1], 10, 64)
			}
		}

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(status)

		const chunkSize = 1024 * 1024
		offset := start - (start % chunkSize)

		for offset <= end {
			result, err := api.UploadGetFile(r.Context(), &tg.UploadGetFileRequest{
				Location: loc,
				Offset:   offset,
				Limit:    chunkSize,
			})
			if err != nil {
				log.Println("download error:", err)
				return
			}

			file, ok := result.(*tg.UploadFile)
			if !ok {
				log.Println("unexpected result type")
				return
			}

			chunkStart := int64(0)
			if offset < start {
				chunkStart = start - offset
			}
			chunkEnd := int64(len(file.Bytes))
			if offset+int64(len(file.Bytes)) > end+1 {
				chunkEnd = end + 1 - offset
			}
			if chunkStart < chunkEnd {
				if _, werr := w.Write(file.Bytes[chunkStart:chunkEnd]); werr != nil {
					return
				}
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			offset += chunkSize
		}
	})

	go func() {
		if err := http.ListenAndServe("127.0.0.1:8080", mux); err != nil {
			log.Println("server error:", err)
		}
	}()

	return "http://127.0.0.1:8080/play"
}
