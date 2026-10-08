package tts

import (
	"bytes"
	"io"
	"sync"
)

// A single cache entry is capped too, so an unusually long username cannot
// grow a temporary capture without bound.
const cacheLimit = 8 << 20
const entryLimit = 256 << 10

type captureWriter struct {
	dst         io.Writer
	cache, seen bool
	data        []byte
}

func (w *captureWriter) Write(p []byte) (int, error) {
	if len(p) > 0 && !w.seen {
		w.seen = true
	}
	if w.cache {
		if len(w.data)+len(p) > entryLimit {
			w.cache = false
			w.data = nil
		} else {
			w.data = append(w.data, p...)
		}
	}
	return w.dst.Write(p)
}

type prefixEntry struct {
	key  string
	data []byte
}
type prefixCache struct {
	mu      sync.Mutex
	entries []prefixEntry
	size    int
}

func (c *prefixCache) get(key string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, e := range c.entries {
		if e.key == key {
			copy(c.entries[i:], c.entries[i+1:])
			c.entries[len(c.entries)-1] = e
			return e.data
		}
	}
	return nil
}
func (c *prefixCache) put(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	size := len(key) + len(data)
	if len(data) == 0 || size > entryLimit {
		return
	}
	for _, e := range c.entries {
		if e.key == key {
			return
		}
	}
	for c.size+size > cacheLimit {
		e := c.entries[0]
		c.size -= len(e.key) + len(e.data)
		c.entries[0] = prefixEntry{}
		c.entries = c.entries[1:]
	}
	data = bytes.Clone(data)
	c.entries = append(c.entries, prefixEntry{key, data})
	c.size += size
}
