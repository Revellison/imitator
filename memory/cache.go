package memory

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Cache is a thread-safe in-memory cache for media chunks
type Cache struct {
	mu     sync.RWMutex
	chunks map[string][]byte
	keys   []string // For random access
}

func NewCache() *Cache {
	return &Cache{
		chunks: make(map[string][]byte),
		keys:   make([]string, 0),
	}
}

// LoadFromDirectory preloads all .ts files from a given directory into memory
func (c *Cache) LoadFromDirectory(dir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ts") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			continue
		}

		name := entry.Name()
		// Try to extract ID from name like "123.ts"
		id := strings.TrimSuffix(name, ".ts")

		c.chunks[id] = data
		c.keys = append(c.keys, id)
		count++
	}

	log.Printf("Preloaded %d chunks into RAM from %s", count, dir)
	return nil
}

// Get safely retrieves a chunk by string ID
func (c *Cache) Get(id string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, exists := c.chunks[id]
	return data, exists
}

// GetRandom retrieves a random chunk from the preloaded RAM
func (c *Cache) GetRandom(randInt func(int) int) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.keys) == 0 {
		return nil
	}

	idx := randInt(len(c.keys))
	key := c.keys[idx]
	return c.chunks[key]
}
