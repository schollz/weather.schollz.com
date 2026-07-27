package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPersistentCacheRoundTripAndExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	first, err := Open(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Put("fresh", map[string]string{"value": "yes"}, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := first.Put("expired", "old", time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := Open(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	var fresh map[string]string
	if !second.Get("fresh", &fresh) || fresh["value"] != "yes" {
		t.Fatalf("persistent value missing: %#v", fresh)
	}
	var expired string
	if second.Get("expired", &expired) {
		t.Fatalf("expired value was returned: %q", expired)
	}
}

func TestMemoryCacheIsBounded(t *testing.T) {
	cache := NewMemory(2)
	for _, key := range []string{"one", "two", "three"} {
		if err := cache.Put(key, key, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if len(cache.memory) != 2 {
		t.Fatalf("memory cache grew to %d entries", len(cache.memory))
	}
}
