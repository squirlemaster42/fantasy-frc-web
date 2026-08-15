package draft

import (
	"container/list"
	"sync"
)

// lruCache is a simple thread-safe LRU cache backed by the standard library's
// container/list and a map. It is generic so it can be reused, but the draft
// package uses it as lruCache[int, *DraftActor].
type lruCache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	order    *list.List
	onEvict  func(K, V)
	mu       sync.Mutex
}

type cacheEntry[K comparable, V any] struct {
	key   K
	value V
}

func newLRUCache[K comparable, V any](capacity int, onEvict func(K, V)) *lruCache[K, V] {
	if capacity < 1 {
		capacity = 1
	}
	return &lruCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		order:    list.New(),
		onEvict:  onEvict,
	}
}

func (c *lruCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	c.order.MoveToFront(elem)
	return elem.Value.(*cacheEntry[K, V]).value, true
}

func (c *lruCache[K, V]) Add(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*cacheEntry[K, V]).value = value
		return
	}

	elem := c.order.PushFront(&cacheEntry[K, V]{key: key, value: value})
	c.items[key] = elem

	if c.order.Len() > c.capacity {
		c.evictOldest()
	}
}

func (c *lruCache[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return false
	}

	c.removeElement(elem)
	return true
}

func (c *lruCache[K, V]) Contains(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.items[key]
	return ok
}

func (c *lruCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}

func (c *lruCache[K, V]) evictOldest() {
	elem := c.order.Back()
	if elem == nil {
		return
	}

	c.removeElement(elem)
}

func (c *lruCache[K, V]) removeElement(elem *list.Element) {
	entry := elem.Value.(*cacheEntry[K, V])
	delete(c.items, entry.key)
	c.order.Remove(elem)

	if c.onEvict != nil {
		c.onEvict(entry.key, entry.value)
	}
}
