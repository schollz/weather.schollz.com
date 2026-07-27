package store

import (
	"context"
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
	defer cache.Close()
	for _, key := range []string{"one", "two", "three"} {
		if err := cache.Put(key, key, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if len(cache.memory) != 2 {
		t.Fatalf("memory cache grew to %d entries", len(cache.memory))
	}
}

func TestPersistentCacheAttachesAfterRollingDeployLockIsReleased(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	first, err := open(path, 4, 20*time.Millisecond, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Put("existing", "persisted", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	second, err := open(path, 4, 20*time.Millisecond, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if second.Persistent() {
		t.Fatal("replacement unexpectedly acquired the locked database")
	}
	if err := second.Put("during-handoff", "memory", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := second.WaitForPersistence(ctx); err != nil {
		t.Fatalf("replacement did not attach persistent cache: %v", err)
	}
	if !second.Persistent() {
		t.Fatal("replacement cache is not persistent")
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}

	third, err := open(path, 4, 20*time.Millisecond, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer third.Close()
	for key, expected := range map[string]string{
		"existing":       "persisted",
		"during-handoff": "memory",
	} {
		var actual string
		if !third.Get(key, &actual) || actual != expected {
			t.Fatalf("%s = %q, want %q", key, actual, expected)
		}
	}
}
