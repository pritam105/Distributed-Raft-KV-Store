package kv

import (
	"encoding/json"
	"log"

	"distributed-raft-kv-store/storage"
)

// Apply executes a single storage.Entry against the store.
func (s *Store) Apply(entry storage.Entry) error {
	if entry.Key == "" {
		return ErrEmptyKey
	}

	switch entry.Op {
	case storage.OpDelete:
		return s.Delete(entry.Key)
	case storage.OpUpsert:
		return s.Upsert(entry.Key, entry.Value)
	default:
		return storage.ErrUnknownOperation
	}
}

// RunApplyLoop reads committed commands from applyCh and applies them to the store.
// Each command is a JSON-encoded storage.Entry. Blocks until the channel is closed.
func RunApplyLoop(cmdCh <-chan []byte, store *Store) {
	for cmd := range cmdCh {
		var entry storage.Entry
		if err := json.Unmarshal(cmd, &entry); err != nil {
			log.Printf("[apply] failed to decode command: %v", err)
			continue
		}
		if err := store.Apply(entry); err != nil {
			log.Printf("[apply] failed to apply %s %s: %v", entry.Op, entry.Key, err)
		}
	}
}
