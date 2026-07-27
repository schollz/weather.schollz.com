package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDirectoryDefaultsToGoTempDirectory(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	expected := filepath.Join(os.TempDir(), "wthrtxt")
	if actual := cacheDirectory(); actual != expected {
		t.Fatalf("got %q, want %q", actual, expected)
	}
}

func TestCacheDirectoryHonorsContainerConfiguration(t *testing.T) {
	t.Setenv("DATA_DIR", "/data")
	if actual := cacheDirectory(); actual != "/data" {
		t.Fatalf("got %q, want /data", actual)
	}
}
