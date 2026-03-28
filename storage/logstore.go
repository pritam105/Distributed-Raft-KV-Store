package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Operation string

const (
	OpUpsert Operation = "upsert"
	OpDelete Operation = "delete"
)

var ErrUnknownOperation = errors.New("unknown operation")

type Entry struct {
	Op    Operation `json:"op"`
	Key   string    `json:"key"`
	Value string    `json:"value,omitempty"`
}

type WAL interface {
	Append(entry Entry) error
	Load() ([]Entry, error)
	Close() error
}

type FileWAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func NewFileWAL(path string) (*FileWAL, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	return &FileWAL{
		file: file,
		path: path,
	}, nil
}

func (w *FileWAL) Append(entry Entry) error {
	if entry.Op != OpUpsert && entry.Op != OpDelete {
		return ErrUnknownOperation
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if _, err := w.file.Write(append(payload, '\n')); err != nil {
		return err
	}

	return w.file.Sync()
}

func (w *FileWAL) Load() ([]Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(w.file)
	entries := make([]Entry, 0)
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		if entry.Op != OpUpsert && entry.Op != OpDelete {
			return nil, ErrUnknownOperation
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	_, err := w.file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (w *FileWAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

type NoopWAL struct{}

func NewNoopWAL() NoopWAL {
	return NoopWAL{}
}

func (NoopWAL) Append(Entry) error {
	return nil
}

func (NoopWAL) Load() ([]Entry, error) {
	return nil, nil
}

func (NoopWAL) Close() error {
	return nil
}
