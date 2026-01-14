package cache

import (
	"testing"
	"time"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	c := NewLRUCache(3, 1*time.Minute)

	// Set and Get
	c.Set("key1", []byte("value1"))
	if got := c.Get("key1"); string(got) != "value1" {
		t.Errorf("expected value1, got %s", string(got))
	}

	// Miss
	if got := c.Get("nonexistent"); got != nil {
		t.Errorf("expected nil for nonexistent key, got %v", got)
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	c := NewLRUCache(3, 1*time.Minute)

	c.Set("key1", []byte("value1"))
	c.Set("key2", []byte("value2"))
	c.Set("key3", []byte("value3"))

	// Access key1 to make it recently used
	c.Get("key1")

	// Add key4 - should evict key2 (least recently used)
	c.Set("key4", []byte("value4"))

	if got := c.Get("key2"); got != nil {
		t.Errorf("key2 should have been evicted, got %s", string(got))
	}

	if got := c.Get("key1"); string(got) != "value1" {
		t.Errorf("key1 should still exist, got %s", string(got))
	}
}

func TestLRUCache_TTLExpiration(t *testing.T) {
	c := NewLRUCache(10, 50*time.Millisecond)

	c.Set("key1", []byte("value1"))

	// Should exist immediately
	if got := c.Get("key1"); string(got) != "value1" {
		t.Errorf("expected value1, got %s", string(got))
	}

	// Wait for TTL
	time.Sleep(60 * time.Millisecond)

	// Should be expired
	if got := c.Get("key1"); got != nil {
		t.Errorf("key1 should have expired, got %s", string(got))
	}
}

func TestLRUCache_Stats(t *testing.T) {
	c := NewLRUCache(10, 1*time.Minute)

	c.Set("key1", []byte("value1"))

	// 2 hits
	c.Get("key1")
	c.Get("key1")

	// 1 miss
	c.Get("nonexistent")

	stats := c.Stats()
	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.Size != 1 {
		t.Errorf("expected size 1, got %d", stats.Size)
	}
}

func TestLRUCache_Clear(t *testing.T) {
	c := NewLRUCache(10, 1*time.Minute)

	c.Set("key1", []byte("value1"))
	c.Set("key2", []byte("value2"))

	c.Clear()

	if c.Len() != 0 {
		t.Errorf("expected empty cache, got %d", c.Len())
	}
}

func TestLRUCache_Update(t *testing.T) {
	c := NewLRUCache(10, 1*time.Minute)

	c.Set("key1", []byte("value1"))
	c.Set("key1", []byte("value2"))

	if got := c.Get("key1"); string(got) != "value2" {
		t.Errorf("expected value2, got %s", string(got))
	}

	if c.Len() != 1 {
		t.Errorf("expected size 1, got %d", c.Len())
	}
}

func TestHashKey(t *testing.T) {
	key1 := HashKey("model", "request1")
	key2 := HashKey("model", "request2")
	key3 := HashKey("model", "request1")

	if key1 == key2 {
		t.Error("different inputs should produce different keys")
	}

	if key1 != key3 {
		t.Error("same inputs should produce same key")
	}

	if len(key1) != 32 {
		t.Errorf("expected key length 32, got %d", len(key1))
	}
}
