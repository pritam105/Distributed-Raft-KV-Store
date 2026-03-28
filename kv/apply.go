package kv

import "distributed-raft-kv-store/storage"

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
