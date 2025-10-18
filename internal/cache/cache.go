package cache

import (
	"container/list"
	"sync"
)

//go:generate moq -pkg mocks -out ../mocks/cache_mock.go . Store

// Store describes cache operations used by the service.
type Store interface {
	Get(id string) ([]byte, bool)
	Set(id string, b []byte)
}

type entry struct {
	key   string
	value []byte
}

type Cache struct {
	mu    sync.Mutex
	items map[string]*list.Element
	order *list.List
	limit int
}

func New(limit int) *Cache {
	if limit <= 0 {
		limit = 1000
	}
	return &Cache{
		items: make(map[string]*list.Element, limit),
		order: list.New(),
		limit: limit,
	}
}

func (c *Cache) Get(id string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[id]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(elem)
	ent := elem.Value.(*entry)
	return ent.value, true
}

func (c *Cache) Set(id string, b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		c.order.MoveToFront(elem)
		buf := make([]byte, len(b))
		copy(buf, b)
		elem.Value.(*entry).value = buf
		return
	}

	buf := make([]byte, len(b))
	copy(buf, b)
	elem := c.order.PushFront(&entry{key: id, value: buf})
	c.items[id] = elem

	if len(c.items) > c.limit {
		c.evict()
	}
}

func (c *Cache) evict() {
	tail := c.order.Back()
	if tail == nil {
		return
	}
	ent := tail.Value.(*entry)
	delete(c.items, ent.key)
	c.order.Remove(tail)
}

func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

var _ Store = (*Cache)(nil)
