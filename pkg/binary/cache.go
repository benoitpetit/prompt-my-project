package binary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Cache manages a persistent cache of binary file detection results
type Cache struct {
	sync.RWMutex
	cache     map[string]bool
	cacheDir  string
	cacheFile string
}

// NewCache creates a new cache for binary files
func NewCache() *Cache {
	homeDir, err := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".pmp", "cache")
	if err != nil {
		// In case of error, use a temporary directory
		cacheDir = filepath.Join(os.TempDir(), "pmp-cache")
	}

	return &Cache{
		cache:     make(map[string]bool),
		cacheDir:  cacheDir,
		cacheFile: filepath.Join(cacheDir, "binary_cache.json"),
	}
}

// Load loads the cache from disk
func (bc *Cache) Load() error {
	bc.Lock()
	defer bc.Unlock()

	// Check if cache file exists
	if _, err := os.Stat(bc.cacheFile); os.IsNotExist(err) {
		// Create cache directory if needed
		if err := os.MkdirAll(bc.cacheDir, 0755); err != nil {
			return fmt.Errorf("error creating cache directory: %w", err)
		}
		// No existing cache, nothing to load
		return nil
	}

	// Read cache file
	data, err := os.ReadFile(bc.cacheFile)
	if err != nil {
		return fmt.Errorf("error reading cache file: %w", err)
	}

	// Deserialize the cache
	if err := json.Unmarshal(data, &bc.cache); err != nil {
		// Reset cache on error
		bc.cache = make(map[string]bool)
		return fmt.Errorf("error parsing cache file: %w", err)
	}

	return nil
}

// Save saves the cache to disk
func (bc *Cache) Save() error {
	bc.RLock()
	defer bc.RUnlock()

	// Create cache directory if needed
	if err := os.MkdirAll(bc.cacheDir, 0755); err != nil {
		return fmt.Errorf("error creating cache directory: %w", err)
	}

	// Serialize the cache
	data, err := json.MarshalIndent(bc.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing cache: %w", err)
	}

	// Write cache file
	if err := os.WriteFile(bc.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("error writing cache file: %w", err)
	}

	return nil
}

// Get retrieves a value from the cache
func (bc *Cache) Get(key string) (bool, bool) {
	bc.RLock()
	defer bc.RUnlock()
	val, ok := bc.cache[key]
	return val, ok
}

// Set adds a value to the cache
func (bc *Cache) Set(key string, value bool) {
	bc.Lock()
	defer bc.Unlock()
	bc.cache[key] = value
} 