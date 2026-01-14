package cache

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequestDeduplicator_Basic(t *testing.T) {
	d := NewRequestDeduplicator()

	result, err := d.Do("key1", func() ([]byte, error) {
		return []byte("result1"), nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(result) != "result1" {
		t.Errorf("expected result1, got %s", string(result))
	}
}

func TestRequestDeduplicator_Concurrent(t *testing.T) {
	d := NewRequestDeduplicator()

	var callCount int32
	var wg sync.WaitGroup

	// Launch 10 goroutines with the same key
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Do("same-key", func() ([]byte, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(50 * time.Millisecond)
				return []byte("result"), nil
			})
		}()
	}

	wg.Wait()

	// Function should only be called once
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRequestDeduplicator_DifferentKeys(t *testing.T) {
	d := NewRequestDeduplicator()

	var callCount int32
	var wg sync.WaitGroup

	// Launch 3 goroutines with different keys
	for i := 0; i < 3; i++ {
		wg.Add(1)
		key := string(rune('a' + i))
		go func(k string) {
			defer wg.Done()
			d.Do(k, func() ([]byte, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(20 * time.Millisecond)
				return []byte(k), nil
			})
		}(key)
	}

	wg.Wait()

	// Each key should trigger its own call
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestResponseCache_Basic(t *testing.T) {
	cfg := ResponseCacheConfig{
		MaxEntries: 100,
		TTLSeconds: 60,
	}
	c := NewResponseCache(cfg)

	c.Set("model1", "hash1", []byte("response1"))

	got := c.Get("model1", "hash1")
	if string(got) != "response1" {
		t.Errorf("expected response1, got %s", string(got))
	}

	// Different model or hash should miss
	if got := c.Get("model2", "hash1"); got != nil {
		t.Errorf("expected nil for different model, got %v", got)
	}
}

func TestResponseCache_EmptyNotCached(t *testing.T) {
	c := NewResponseCache(DefaultResponseCacheConfig())

	c.Set("model1", "hash1", []byte{})

	if got := c.Get("model1", "hash1"); got != nil {
		t.Errorf("empty responses should not be cached, got %v", got)
	}
}
