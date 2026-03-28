package kv

import (
	"errors"
	"sync"

	"distributed-raft-kv-store/storage"
)

var ErrEmptyKey = errors.New("key cannot be empty")

type Store struct {
	mu   sync.RWMutex
	data map[string]string
	wal  storage.WAL
}

func NewStore(wal storage.WAL) *Store {
	return &Store{
		data: make(map[string]string),
		wal:  wal,
	}
}

func NewStoreFromWAL(wal storage.WAL) (*Store, error) {
	store := NewStore(wal)
	if wal == nil {
		return store, nil
	}

	entries, err := wal.Load()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		store.apply(entry)
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

	if err := s.append(entry); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyLocked(entry)
	return nil
}

func (s *Store) Delete(key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	entry := storage.Entry{
		Op:  storage.OpDelete,
		Key: key,
	}

	if err := s.append(entry); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyLocked(entry)
	return nil
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
