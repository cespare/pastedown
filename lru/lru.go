// Package lru provides a simple LRU cache that keys []byte values by strings.
package lru

import (
	"sync"
)

type cacheValue struct {
	key        string
	val        []byte
	next, prev *cacheValue
}

func (v *cacheValue) size() int64 {
	return int64(len([]byte(v.key)) + len(v.val))
}

type cacheValueList struct {
	front *cacheValue
	back  *cacheValue
}

func (l *cacheValueList) pushFront(v *cacheValue) {
	v.next = l.front
	v.prev = nil
	if l.front == nil {
		l.back = v
	} else {
		l.front.prev = v
	}
	l.front = v
}

func (l *cacheValueList) moveToFront(v *cacheValue) {
	if v.prev == nil {
		return
	}
	v.prev.next = v.next
	if v.next == nil {
		l.back = v.prev
	} else {
		v.next.prev = v.prev
	}
	v.prev = nil
	v.next = l.front
	if l.front != nil {
		l.front.prev = v
	}
	l.front = v
}

func (l *cacheValueList) delete(v *cacheValue) {
	if v.prev == nil {
		l.front = v.next
	} else {
		v.prev.next = v.next
	}
	if v.next == nil {
		l.back = v.prev
	} else {
		v.next.prev = v.prev
	}
}

// A Cache is a size-bounded LRU cache which associates string keys
// with []byte values.
// All methods are safe for concurrent use by multiple goroutines.
type Cache struct {
	mu sync.Mutex

	size     int64
	capacity int64
	list     cacheValueList
	table    map[string]*cacheValue
}

// New creates a new Cache with a maximum size of capacity bytes.
func New(capacity int64) *Cache {
	if capacity < 0 {
		panic("lru: bad capacity")
	}
	return &Cache{
		capacity: capacity,
		table:    make(map[string]*cacheValue),
	}
}

// Insert adds val to the cache indexed by the given key after evicting enough
// existing items (least recently used first) to keep the total size beneath
// this cache's capacity.
// It does not do anything if the key is already present in the cache or if the
// size of this single item is greater than the cache's capacity.
func (c *Cache) Insert(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.table[key]; ok {
		return
	}
	v := &cacheValue{key: key, val: val}
	if v.size() > c.capacity {
		return
	}
	for c.size+v.size() > c.capacity {
		c.size -= c.list.back.size()
		delete(c.table, c.list.back.key)
		c.list.delete(c.list.back)
	}
	c.list.pushFront(v)
	c.table[key] = v
	c.size += v.size()
}

// Get retrieves a value from the cache by key and indicates
// whether an item was found.
func (c *Cache) Get(key string) (val []byte, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.table[key]
	if !ok {
		return nil, false
	}
	c.list.moveToFront(v)
	return v.val, true
}

// Delete removes the item indicated by the key, if it is present.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.table[key]
	if !ok {
		return
	}
	delete(c.table, key)
	c.list.delete(v)
	c.size -= v.size()
}
