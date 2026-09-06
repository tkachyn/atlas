package store

import (
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
}

// describe one live key and its expiration
type SnapshotEntry struct {
	Key       string
	Value     string
	ExpiresAt time.Time
}

// store provides concurrent access to atlas key value data
type Store struct {
	mu      sync.Mutex
	entries map[string]entry
}

func New() *Store {
	return &Store{entries: make(map[string]entry)}
}

// set stores a value without an expiration
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = entry{value: value}
}

// get returns a value when the key exists and has not expired
func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.entries[key]
	if !ok {
		return "", false
	}

	if s.expired(key, item) {
		return "", false
	}

	return item.value, true
}

// delete removes a key and reports whether it existed
func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.entries[key]
	if !ok {
		return false
	}

	if s.expired(key, item) {
		return false
	}

	delete(s.entries, key)
	return true
}

// exists reports whether a key exists and has not expired
func (s *Store) Exists(key string) bool {
	_, ok := s.Get(key)
	return ok
}

// expire sets a lifetime in seconds and reports whether the key exists
func (s *Store) Expire(key string, seconds int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.expireAt(key, time.Now().Add(time.Duration(seconds)*time.Second))
}

// set an absolute expiration time and report whether the key exists
func (s *Store) ExpireAt(key string, expiresAt time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.expireAt(key, expiresAt)
}

func (s *Store) expireAt(key string, expiresAt time.Time) bool {
	item, ok := s.entries[key]
	if !ok || s.expired(key, item) {
		return false
	}

	item.expiresAt = expiresAt
	s.entries[key] = item
	return true
}

// ttl returns remaining seconds, -1 for no expiration, or -2 for missing keys
func (s *Store) TTL(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.entries[key]
	if !ok {
		return -2
	}

	if s.expired(key, item) {
		return -2
	}

	if item.expiresAt.IsZero() {
		return -1
	}

	return int64(time.Until(item.expiresAt) / time.Second)
}

// snapshot returns all live entries for persistence compaction
func (s *Store) Snapshot() []SnapshotEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := make([]SnapshotEntry, 0, len(s.entries))
	for key, item := range s.entries {
		if s.expired(key, item) {
			continue
		}
		snapshot = append(snapshot, SnapshotEntry{
			Key:       key,
			Value:     item.value,
			ExpiresAt: item.expiresAt,
		})
	}

	return snapshot
}

func (s *Store) expired(key string, item entry) bool {
	if item.expiresAt.IsZero() || time.Now().Before(item.expiresAt) {
		return false
	}

	delete(s.entries, key)
	return true
}
