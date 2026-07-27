package store

import (
	"context"
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
	dbMu       sync.RWMutex
	db         *bolt.DB
	path       string
	ready      chan struct{}
	stop       chan struct{}
	closeOnce  sync.Once
	attachErr  error
	maxEntries int
	mu         sync.Mutex
	memory     map[string]entry
}

func Open(path string, maxEntries int) (*Store, error) {
	return open(path, maxEntries, 2*time.Second, 250*time.Millisecond)
}

func open(path string, maxEntries int, lockTimeout, retryDelay time.Duration) (*Store, error) {
	if maxEntries < 1 {
		maxEntries = 256
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	store := &Store{
		path:       path,
		ready:      make(chan struct{}),
		stop:       make(chan struct{}),
		maxEntries: maxEntries,
		memory:     make(map[string]entry),
	}

	db, err := openPersistent(path, lockTimeout)
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			go store.attachWhenAvailable(lockTimeout, retryDelay)
			return store, nil
		}
		close(store.ready)
		return nil, err
	}

	store.db = db
	close(store.ready)
	return store, nil
}

func openPersistent(path string, timeout time.Duration) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: timeout})
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
	return db, nil
}

func (s *Store) attachWhenAvailable(lockTimeout, retryDelay time.Duration) {
	defer close(s.ready)

	for {
		select {
		case <-s.stop:
			s.setAttachError(context.Canceled)
			return
		default:
		}

		db, err := openPersistent(s.path, lockTimeout)
		if err == nil {
			s.attach(db)
			return
		}
		if !errors.Is(err, bolt.ErrTimeout) {
			s.setAttachError(err)
			return
		}

		timer := time.NewTimer(retryDelay)
		select {
		case <-s.stop:
			if !timer.Stop() {
				<-timer.C
			}
			s.setAttachError(context.Canceled)
			return
		case <-timer.C:
		}
	}
}

func (s *Store) attach(db *bolt.DB) {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	select {
	case <-s.stop:
		s.attachErr = context.Canceled
		_ = db.Close()
		return
	default:
	}

	s.mu.Lock()
	pending := make(map[string]entry, len(s.memory))
	for key, value := range s.memory {
		pending[key] = value
	}
	s.mu.Unlock()

	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(cacheBucket)
		for key, value := range pending {
			encoded, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				return marshalErr
			}
			if putErr := bucket.Put([]byte(key), encoded); putErr != nil {
				return putErr
			}
		}
		return nil
	})
	if err != nil {
		s.attachErr = err
		_ = db.Close()
		return
	}
	s.db = db
}

func (s *Store) setAttachError(err error) {
	s.dbMu.Lock()
	s.attachErr = err
	s.dbMu.Unlock()
}

func (s *Store) Persistent() bool {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
	return s.db != nil
}

func (s *Store) WaitForPersistence(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ready:
	}

	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
	if s.db != nil {
		return nil
	}
	if s.attachErr != nil {
		return s.attachErr
	}
	return errors.New("persistent cache is unavailable")
}

func NewMemory(maxEntries int) *Store {
	if maxEntries < 1 {
		maxEntries = 256
	}
	ready := make(chan struct{})
	close(ready)
	return &Store{
		ready:      ready,
		stop:       make(chan struct{}),
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

	s.dbMu.RLock()
	if s.db == nil {
		s.dbMu.RUnlock()
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
	s.dbMu.RUnlock()
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

	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
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

	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
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
	var closeErr error
	s.closeOnce.Do(func() {
		close(s.stop)
		<-s.ready

		s.dbMu.Lock()
		defer s.dbMu.Unlock()
		if s.db != nil {
			closeErr = s.db.Close()
			s.db = nil
		}
	})
	return closeErr
}
