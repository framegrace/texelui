package textrender

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseCellSizeResponse(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantWidth   int
		wantHeight  int
		wantErr     bool
	}{
		{
			name:       "standard",
			input:      "\x1b[6;20;10t",
			wantWidth:  10,
			wantHeight: 20,
		},
		{
			name:       "typical",
			input:      "\x1b[6;16;8t",
			wantWidth:  8,
			wantHeight: 16,
		},
		{
			name:    "malformed",
			input:   "\x1b[6;t",
			wantErr: true,
		},
		{
			name:    "wrong code",
			input:   "\x1b[5;16;8t",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, h, err := parseCellSizeResponse([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got width=%d height=%d", w, h)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w != tc.wantWidth {
				t.Errorf("width: got %d, want %d", w, tc.wantWidth)
			}
			if h != tc.wantHeight {
				t.Errorf("height: got %d, want %d", h, tc.wantHeight)
			}
		})
	}
}

func TestQueryCellSize_Mock(t *testing.T) {
	var writer bytes.Buffer
	reader := strings.NewReader("\x1b[6;16;8t")

	w, h, err := QueryCellSize(&writer, reader, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 8 {
		t.Errorf("width: got %d, want 8", w)
	}
	if h != 16 {
		t.Errorf("height: got %d, want 16", h)
	}

	query := writer.String()
	if query != "\x1b[16t" {
		t.Errorf("query written: got %q, want %q", query, "\x1b[16t")
	}
}
