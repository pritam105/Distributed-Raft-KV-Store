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
