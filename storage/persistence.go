package storage

type Config struct {
	Enabled bool
	Path    string
}

func OpenWAL(cfg Config) (WAL, error) {
	if !cfg.Enabled {
		return NewNoopWAL(), nil
	}

	return NewFileWAL(cfg.Path)
}

func OpenSnapshot(cfg Config) (SnapshotStore, error) {
	if !cfg.Enabled {
		return NewNoopSnapshot(), nil
	}

	return NewFileSnapshot(cfg.Path)
}
