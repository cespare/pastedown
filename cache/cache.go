// A simple LRU cache for storing documents ([]byte). When the size maximum is reached, items are evicted
// starting with the least recently used. This data structure is goroutine-safe (it has a lock around all
// operations).
package cache

import (
	"container/list"
	"sync"
)

type cacheValue struct {
	key string
	bytes []byte
}

// Just an estimate
func (v *cacheValue) size() uint64 {
	return uint64(len([]byte(v.key)) + len(v.bytes))
}

type Cache struct {
	sync.Mutex

	// The approximate size of the structure (doesn't include the overhead of the data structures; just the
	// sum of the size of the stored documents).
	Size uint64

	capacity uint64
	list     *list.List
	table    map[string]*list.Element
}

// Create a new Cache with a maximum size of capacity bytes.
func New(capacity uint64) *Cache {
	return &Cache{
		capacity: capacity,
		list:     list.New(),
		table:    make(map[string]*list.Element),
	}
}

// Insert some {key, document} into the cache. Doesn't do anything if the key is already present.
func (c *Cache) Insert(key string, document []byte) {
	c.Lock()
	defer c.Unlock()

	_, ok := c.table[key]
	if ok {
		return
	}
	v := &cacheValue{key, document}
	elt := c.list.PushFront(v)
	c.table[key] = elt
	c.Size += v.size()
	c.trim()
}

// Get retrieves a value from the cache and returns the value and an indicator boolean to show whether it was
// present.
func (c *Cache) Get(key string) (document []byte, ok bool) {
	c.Lock()
	defer c.Unlock()

	elt, ok := c.table[key]
	if !ok {
		return nil, false
	}
	c.list.MoveToFront(elt)
	return elt.Value.(*cacheValue).bytes, true
}

// If the key is present, move that document to the front of the list to show that it was most recently used.
func (c *Cache) Update(key string) {
	c.Lock()
	defer c.Unlock()

	elt, ok := c.table[key]
	if !ok {
		return
	}
	c.list.MoveToFront(elt)
}

// Delete the document indicated by the key, if it is present.
func (c *Cache) Delete(key string) {
	c.Lock()
	defer c.Unlock()

	elt, ok := c.table[key]
	if !ok {
		return
	}
	delete(c.table, key)
	v := c.list.Remove(elt).(*cacheValue)
	c.Size -= v.size()
}

// If the cache is over capacity, clear elements (starting at the end of the list) until it is back under
// capacity. Note that this method is not threadsafe (it should only be called from other methods which
// already hold the lock).
func (c *Cache) trim() {
	for c.Size > c.capacity {
		elt := c.list.Back()
		if elt == nil {
			return
		}
		v := c.list.Remove(elt).(*cacheValue)
		delete(c.table, v.key)
		c.Size -= v.size()
	}
}
