package cache_test

import (
	"testing"

	"github.com/kosovrzn/wb-tech-l0/internal/cache"
)

func TestCacheEvictionRespectsLRU(t *testing.T) {
	c := cache.New(2)

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))

	if _, ok := c.Get("a"); !ok {
		t.Fatalf("expected to find key a")
	}

	c.Set("c", []byte("3")) // should evict b

	if _, ok := c.Get("b"); ok {
		t.Fatalf("expected key b to be evicted")
	}
	if v, ok := c.Get("a"); !ok || string(v) != "1" {
		t.Fatalf("expected key a to remain with value 1, got %q ok=%v", v, ok)
	}
	if v, ok := c.Get("c"); !ok || string(v) != "3" {
		t.Fatalf("expected key c with value 3")
	}
}

func TestCacheUpdateKeepsEntry(t *testing.T) {
	c := cache.New(2)

	c.Set("x", []byte("1"))
	c.Set("y", []byte("2"))
	c.Set("x", []byte("10"))

	if c.Len() != 2 {
		t.Fatalf("expected len 2 got %d", c.Len())
	}
	if v, ok := c.Get("x"); !ok || string(v) != "10" {
		t.Fatalf("expected updated value for x")
	}
	c.Set("z", []byte("3"))
	if _, ok := c.Get("y"); ok {
		t.Fatalf("expected y to be evicted")
	}
}
