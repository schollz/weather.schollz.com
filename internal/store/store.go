package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

var cacheBucket = []byte("cache")

type entry struct {
	Data      json.RawMessage `json:"data"`
	ExpiresAt time.Time       `json:"expires_at"`
	StoredAt  time.Time       `json:"stored_at"`
}

// Store is a small bounded memory cache backed by BoltDB. The persistent layer
// is optional so tests and degraded local development can run in memory.
type Store struct {
	db         *bolt.DB
	maxEntries int
	mu         sync.Mutex
	memory     map[string]entry
}

func Open(path string, maxEntries int) (*Store, error) {
	if maxEntries < 1 {
		maxEntries = 256
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		_, createErr := tx.CreateBucketIfNotExists(cacheBucket)
		return createErr
	}); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		db:         db,
		maxEntries: maxEntries,
		memory:     make(map[string]entry),
	}, nil
}

func NewMemory(maxEntries int) *Store {
	if maxEntries < 1 {
		maxEntries = 256
	}
	return &Store{
		maxEntries: maxEntries,
		memory:     make(map[string]entry),
	}
}

func (s *Store) Get(key string, destination any) bool {
	now := time.Now()

	s.mu.Lock()
	cached, found := s.memory[key]
	if found && !cached.ExpiresAt.After(now) {
		delete(s.memory, key)
		found = false
	}
	s.mu.Unlock()

	if found {
		return json.Unmarshal(cached.Data, destination) == nil
	}

	if s.db == nil {
		return false
	}

	var persisted entry
	err := s.db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket(cacheBucket).Get([]byte(key))
		if value == nil {
			return errors.New("not found")
		}
		return json.Unmarshal(value, &persisted)
	})
	if err != nil || !persisted.ExpiresAt.After(now) {
		if err == nil {
			_ = s.Delete(key)
		}
		return false
	}

	s.remember(key, persisted)
	return json.Unmarshal(persisted.Data, destination) == nil
}

func (s *Store) Put(key string, value any, expiresAt time.Time) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	cached := entry{
		Data:      data,
		ExpiresAt: expiresAt,
		StoredAt:  time.Now(),
	}
	s.remember(key, cached)

	if s.db == nil {
		return nil
	}

	encoded, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(cacheBucket).Put([]byte(key), encoded)
	})
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	delete(s.memory, key)
	s.mu.Unlock()

	if s.db == nil {
		return nil
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(cacheBucket).Delete([]byte(key))
	})
}

func (s *Store) remember(key string, value entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.memory) >= s.maxEntries {
		var oldestKey string
		var oldestTime time.Time
		for candidateKey, candidate := range s.memory {
			if oldestKey == "" || candidate.StoredAt.Before(oldestTime) {
				oldestKey = candidateKey
				oldestTime = candidate.StoredAt
			}
		}
		delete(s.memory, oldestKey)
	}
	s.memory[key] = value
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
