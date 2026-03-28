package kv

import (
	"path/filepath"
	"testing"

	"distributed-raft-kv-store/storage"
)

func TestStoreUpsertReadDelete(t *testing.T) {
	store := NewStore(storage.NewNoopWAL(), storage.NewNoopSnapshot())

	if err := store.Upsert("user:1", "alice"); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	value, ok, err := store.Get("user:1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected key to exist")
	}
	if value != "alice" {
		t.Fatalf("expected alice, got %q", value)
	}

	if err := store.Upsert("user:1", "bob"); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	value, ok, err = store.Get("user:1")
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if !ok || value != "bob" {
		t.Fatalf("expected updated value bob, got %q (ok=%v)", value, ok)
	}

	if err := store.Delete("user:1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, ok, err = store.Get("user:1")
	if err != nil {
		t.Fatalf("get after delete failed: %v", err)
	}
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestStoreRestoresFromDisk(t *testing.T) {
	root := t.TempDir()
	walPath := filepath.Join(root, "wal.log")
	snapshotPath := filepath.Join(root, "snapshot.json")

	wal, err := storage.NewFileWAL(walPath)
	if err != nil {
		t.Fatalf("create wal failed: %v", err)
	}
	snapshot, err := storage.NewFileSnapshot(snapshotPath)
	if err != nil {
		t.Fatalf("create snapshot failed: %v", err)
	}

	store := NewStore(wal, snapshot)
	if err := store.Upsert("project", "raft"); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if err := store.Upsert("project", "raft-kv"); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := store.Upsert("owner", "team"); err != nil {
		t.Fatalf("second key failed: %v", err)
	}
	if err := store.Delete("owner"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if err := wal.Close(); err != nil {
		t.Fatalf("close wal failed: %v", err)
	}

	reopened, err := storage.NewFileWAL(walPath)
	if err != nil {
		t.Fatalf("reopen wal failed: %v", err)
	}
	defer reopened.Close()
	reopenedSnapshot, err := storage.NewFileSnapshot(snapshotPath)
	if err != nil {
		t.Fatalf("reopen snapshot failed: %v", err)
	}

	restored, err := NewStoreFromDisk(reopened, reopenedSnapshot)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	value, ok, err := restored.Get("project")
	if err != nil {
		t.Fatalf("restored get failed: %v", err)
	}
	if !ok || value != "raft-kv" {
		t.Fatalf("expected restored value raft-kv, got %q (ok=%v)", value, ok)
	}

	_, ok, err = restored.Get("owner")
	if err != nil {
		t.Fatalf("restored delete check failed: %v", err)
	}
	if ok {
		t.Fatal("expected deleted key to stay deleted after replay")
	}
}
