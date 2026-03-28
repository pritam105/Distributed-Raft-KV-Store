package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type SnapshotStore interface {
	Save(state map[string]string) error
	Load() (map[string]string, error)
}

type FileSnapshot struct {
	mu   sync.Mutex
	path string
}

func NewFileSnapshot(path string) (*FileSnapshot, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	return &FileSnapshot{path: path}, nil
}

func (s *FileSnapshot) Save(state map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

func (s *FileSnapshot) Load() (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	if len(payload) == 0 {
		return map[string]string{}, nil
	}

	state := make(map[string]string)
	if err := json.Unmarshal(payload, &state); err != nil {
		return nil, err
	}

	return state, nil
}

type NoopSnapshot struct{}

func NewNoopSnapshot() NoopSnapshot {
	return NoopSnapshot{}
}

func (NoopSnapshot) Save(map[string]string) error {
	return nil
}

func (NoopSnapshot) Load() (map[string]string, error) {
	return map[string]string{}, nil
}
