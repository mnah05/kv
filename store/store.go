package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"sync"

	tinywal "github.com/mnah05/wal-learn/wal"
)

type Store struct {
	wal          *tinywal.WAL
	data         map[string]string
	mu           sync.RWMutex
	snapshotPath string
}

func Open(walPath string) (*Store, error) {
	wal, err := tinywal.Open(walPath)
	if err != nil {
		return nil, err
	}

	s := &Store{
		wal:          wal,
		data:         map[string]string{},
		snapshotPath: walPath + ".snap",
	}

	s.loadSnapshot()

	if err := s.wal.Replay(func(op byte, key, value string) {
		switch op {
		case tinywal.OpPut:
			s.data[key] = value
		case tinywal.OpDelete:
			delete(s.data, key)
		}
	}); err != nil {
		log.Printf("WAL replay error (partial recovery): %v", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.wal.Close()
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *Store) Put(key, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.wal.Put(key, val); err != nil {
		return err
	}
	s.data[key] = val
	return nil
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.wal.Delete(key); err != nil {
		return err
	}
	delete(s.data, key)
	return nil
}

func (s *Store) Checkpoint() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wal.Checkpoint(s.data)
}

func (s *Store) loadSnapshot() {
	data, err := os.ReadFile(s.snapshotPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Println("snapshot not found — first run, replaying full WAL")
			return
		}
		log.Printf("WARNING: cannot read snapshot (%v), replaying full WAL", err)
		return
	}
	if err := json.Unmarshal(data, &s.data); err != nil {
		log.Printf("WARNING: snapshot corrupt (%v), falling back to WAL replay", err)
		return
	}
	log.Printf("Total number of entries restores: %d", len(s.data))
}
