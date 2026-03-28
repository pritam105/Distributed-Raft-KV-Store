package storage

import (
	"path/filepath"
	"testing"
)

func TestOpenWALDisabledUsesNoop(t *testing.T) {
	wal, err := OpenWAL(Config{Enabled: false})
	if err != nil {
		t.Fatalf("open wal failed: %v", err)
	}
	defer wal.Close()

	if err := wal.Append(Entry{Op: OpUpsert, Key: "a", Value: "1"}); err != nil {
		t.Fatalf("noop wal append failed: %v", err)
	}

	entries, err := wal.Load()
	if err != nil {
		t.Fatalf("noop wal load failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
}

func TestFileWALRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal.log")

	wal, err := OpenWAL(Config{Enabled: true, Path: path})
	if err != nil {
		t.Fatalf("open wal failed: %v", err)
	}
	defer wal.Close()

	entries := []Entry{
		{Op: OpUpsert, Key: "alpha", Value: "1"},
		{Op: OpDelete, Key: "beta"},
	}

	for _, entry := range entries {
		if err := wal.Append(entry); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}

	loaded, err := wal.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(loaded) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(loaded))
	}

	for i := range entries {
		if loaded[i] != entries[i] {
			t.Fatalf("entry %d mismatch: got %+v want %+v", i, loaded[i], entries[i])
		}
	}
}
