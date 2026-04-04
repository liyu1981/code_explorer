package sqlitefs

import (
	"sync"
)

type ChunkCache struct {
	mu    sync.RWMutex
	data  map[ChunkKey][]byte
	order []ChunkKey
	size  int
	max   int
}

func NewChunkCache(max int) *ChunkCache {
	return &ChunkCache{
		data: make(map[ChunkKey][]byte),
		max:  max,
	}
}

func (c *ChunkCache) Get(key ChunkKey) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *ChunkCache) Set(key ChunkKey, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[key]; exists {
		c.removeKey(key)
		c.data[key] = value
		c.order = append(c.order, key)
		c.size++
	} else {
		if c.size >= c.max {
			c.evict()
		}
		c.data[key] = value
		c.order = append(c.order, key)
		c.size++
	}
}

func (c *ChunkCache) removeKey(key ChunkKey) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}

func (c *ChunkCache) evict() {
	if len(c.order) == 0 {
		return
	}
	key := c.order[0]
	delete(c.data, key)
	c.order = c.order[1:]
	c.size--
}
