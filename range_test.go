package main

import (
	"errors"
	"testing"
)

func TestParseRange(t *testing.T) {
	const size = 1000

	tests := []struct {
		name      string
		header    string
		wantStart int64
		wantEnd   int64
		wantErr   error
	}{
		{name: "whole file", header: "", wantStart: 0, wantEnd: 999},
		{name: "open ended", header: "bytes=0-", wantStart: 0, wantEnd: 999},
		{name: "explicit", header: "bytes=100-199", wantStart: 100, wantEnd: 199},
		{name: "start only", header: "bytes=100-", wantStart: 100, wantEnd: 999},
		{name: "suffix", header: "bytes=-500", wantStart: 500, wantEnd: 999},
		{name: "suffix longer than file", header: "bytes=-2000", wantStart: 0, wantEnd: 999},
		{name: "end clamped", header: "bytes=0-999999", wantStart: 0, wantEnd: 999},
		{name: "start at eof", header: "bytes=1000-", wantErr: errRangeNotSatisfiable},
		{name: "start past eof", header: "bytes=2000-3000", wantErr: errRangeNotSatisfiable},
		{name: "reversed", header: "bytes=5-2", wantErr: errRangeNotSatisfiable},
		{name: "multiple ranges", header: "bytes=0-1,3-4", wantErr: errRangeNotSatisfiable},
		{name: "suffix zero", header: "bytes=-0", wantErr: errInvalidRange},
		{name: "bad start", header: "bytes=abc-", wantErr: errInvalidRange},
		{name: "bad end", header: "bytes=1-abc", wantErr: errInvalidRange},
		{name: "wrong unit", header: "items=0-1", wantErr: errInvalidRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseRange(tt.header, size)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("parseRange(%q) error = %v, want %v", tt.header, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRange(%q) error: %v", tt.header, err)
			}
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("parseRange(%q) = %d-%d, want %d-%d", tt.header, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestParseRangeEmptyFile(t *testing.T) {
	if start, end, err := parseRange("", 0); err != nil || start != 0 || end != -1 {
		t.Fatalf("parseRange(empty, 0) = %d-%d, %v; want 0--1, nil", start, end, err)
	}
	if _, _, err := parseRange("bytes=0-", 0); !errors.Is(err, errRangeNotSatisfiable) {
		t.Fatalf("parseRange(range, 0) error = %v, want %v", err, errRangeNotSatisfiable)
	}
}
