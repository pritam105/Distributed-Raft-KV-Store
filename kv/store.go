package kv

import (
	"errors"
	"sync"

	"distributed-raft-kv-store/storage"
)

var ErrEmptyKey = errors.New("key cannot be empty")

const DefaultSnapshotInterval = 100

type Store struct {
	mu               sync.RWMutex
	data             map[string]string
	wal              storage.WAL
	snapshot         storage.SnapshotStore
	writeCount       int
	snapshotInterval int // snapshot every N writes; 0 = disabled
}

func NewStore(wal storage.WAL, snapshot storage.SnapshotStore) *Store {
	return &Store{
		data:             make(map[string]string),
		wal:              wal,
		snapshot:         snapshot,
		snapshotInterval: DefaultSnapshotInterval,
	}
}

func NewStoreFromDisk(wal storage.WAL, snapshot storage.SnapshotStore) (*Store, error) {
	store := NewStore(wal, snapshot)

	if snapshot != nil {
		state, err := snapshot.Load()
		if err != nil {
			return nil, err
		}
		store.data = state
	}

	if wal != nil {
		entries, err := wal.Load()
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			store.apply(entry)
		}
	}

	return store, nil
}

func (s *Store) Upsert(key, value string) error {
	if key == "" {
		return ErrEmptyKey
	}

	entry := storage.Entry{
		Op:    storage.OpUpsert,
		Key:   key,
		Value: value,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistAndApplyLocked(entry)
}

func (s *Store) Delete(key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	entry := storage.Entry{
		Op:  storage.OpDelete,
		Key: key,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistAndApplyLocked(entry)
}

func (s *Store) Get(key string) (string, bool, error) {
	if key == "" {
		return "", false, ErrEmptyKey
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.data[key]
	return value, ok, nil
}

func (s *Store) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]string, len(s.data))
	for key, value := range s.data {
		out[key] = value
	}

	return out
}

func (s *Store) append(entry storage.Entry) error {
	if s.wal == nil {
		return nil
	}

	return s.wal.Append(entry)
}

func (s *Store) apply(entry storage.Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyLocked(entry)
}

func (s *Store) applyLocked(entry storage.Entry) {
	switch entry.Op {
	case storage.OpDelete:
		delete(s.data, entry.Key)
	default:
		s.data[entry.Key] = entry.Value
	}
}

func (s *Store) persistAndApplyLocked(entry storage.Entry) error {
	if err := s.append(entry); err != nil {
		return err
	}

	s.applyLocked(entry)

	if s.snapshot == nil || s.snapshotInterval == 0 {
		return nil
	}

	s.writeCount++
	if s.writeCount%s.snapshotInterval != 0 {
		return nil
	}

	state := make(map[string]string, len(s.data))
	for key, value := range s.data {
		state[key] = value
	}

	return s.snapshot.Save(state)
}
