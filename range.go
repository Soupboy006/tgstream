package main

import (
	"errors"
	"strconv"
	"strings"
)

var (
	errInvalidRange        = errors.New("invalid range")
	errRangeNotSatisfiable = errors.New("range not satisfiable")
)

// parseRange parses a single HTTP byte range for a file of the given size.
// An empty header selects the whole file. Multipart ranges are rejected.
func parseRange(header string, size int64) (start, end int64, err error) {
	if header == "" {
		return 0, size - 1, nil
	}

	spec, ok := strings.CutPrefix(header, "bytes=")
	if !ok {
		return 0, 0, errInvalidRange
	}
	if strings.Contains(spec, ",") {
		return 0, 0, errRangeNotSatisfiable
	}

	first, last, ok := strings.Cut(spec, "-")
	if !ok {
		return 0, 0, errInvalidRange
	}

	if first == "" {
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, errInvalidRange
		}
		if n > size {
			n = size
		}
		if n == 0 {
			return 0, 0, errRangeNotSatisfiable
		}
		return size - n, size - 1, nil
	}

	start, err = strconv.ParseInt(first, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, errInvalidRange
	}
	end = size - 1
	if last != "" {
		end, err = strconv.ParseInt(last, 10, 64)
		if err != nil || end < 0 {
			return 0, 0, errInvalidRange
		}
		if end > size-1 {
			end = size - 1
		}
	}

	if start >= size || start > end {
		return 0, 0, errRangeNotSatisfiable
	}

	return start, end, nil
}
