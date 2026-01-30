package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"prochub/app/config"
)

type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore(baseDir string) *Store {
	return &Store{
		path: filepath.Join(baseDir, "config.json"),
	}
}

func (s *Store) Load() (config.AppConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return config.DefaultConfig(), nil
		}
		return config.AppConfig{}, err
	}

	var cfg config.AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.DefaultConfig(), nil
	}

	return cfg, nil
}

func (s *Store) Save(cfg config.AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o644)
}
