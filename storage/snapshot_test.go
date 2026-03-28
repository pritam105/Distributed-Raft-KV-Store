package storage

import (
	"path/filepath"
	"testing"
)

func TestFileSnapshotRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")

	snapshot, err := OpenSnapshot(Config{Enabled: true, Path: path})
	if err != nil {
		t.Fatalf("open snapshot failed: %v", err)
	}

	state := map[string]string{
		"alpha": "1",
		"beta":  "2",
	}

	if err := snapshot.Save(state); err != nil {
		t.Fatalf("save snapshot failed: %v", err)
	}

	loaded, err := snapshot.Load()
	if err != nil {
		t.Fatalf("load snapshot failed: %v", err)
	}

	if len(loaded) != len(state) {
		t.Fatalf("expected %d keys, got %d", len(state), len(loaded))
	}

	for key, want := range state {
		if got := loaded[key]; got != want {
			t.Fatalf("key %s: got %q want %q", key, got, want)
		}
	}
}

func TestOpenSnapshotDisabledUsesNoop(t *testing.T) {
	snapshot, err := OpenSnapshot(Config{Enabled: false})
	if err != nil {
		t.Fatalf("open noop snapshot failed: %v", err)
	}

	if err := snapshot.Save(map[string]string{"x": "1"}); err != nil {
		t.Fatalf("noop snapshot save failed: %v", err)
	}

	loaded, err := snapshot.Load()
	if err != nil {
		t.Fatalf("noop snapshot load failed: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected empty snapshot, got %d keys", len(loaded))
	}
}
