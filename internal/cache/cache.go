package cache

import "sync"

type Cache struct {
	mu sync.RWMutex
	m  map[string][]byte
}

func New() *Cache { return &Cache{m: make(map[string][]byte)} }

func (c *Cache) Get(id string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[id]
	return v, ok
}

func (c *Cache) Set(id string, b []byte) {
	c.mu.Lock()
	buf := make([]byte, len(b))
	copy(buf, b)
	c.m[id] = buf
	c.mu.Unlock()
}
