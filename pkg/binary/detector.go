package binary

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

// List of extensions to exclude
var BinaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".bin": true, ".obj": true, ".o": true, ".a": true,
	".lib": true, ".pyc": true, ".pyo": true, ".pyd": true,
	".class": true, ".jar": true, ".war": true, ".ear": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".ico": true, ".zip": true, ".tar": true,
	".gz": true, ".rar": true, ".7z": true, ".pdf": true,
}

// IsBinaryFile detects if a file is binary based on its extension, mime type or content
func IsBinaryFile(filepath string, cache *Cache) bool {
	// Get file info for cache key
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return true // When in doubt, consider as binary
	}

	// Create a unique cache key based on path and meta-information
	cacheKey := fmt.Sprintf("%s:%d:%d", filepath, fileInfo.Size(), fileInfo.ModTime().UnixNano())

	// Check in cache
	if cache != nil {
		if isBinary, found := cache.Get(cacheKey); found {
			return isBinary
		}
	}

	// If not found in cache, perform detection
	// Check extension
	ext := strings.ToLower(path.Ext(filepath))
	if BinaryExtensions[ext] {
		if cache != nil {
			cache.Set(cacheKey, true)
		}
		return true
	}

	// Check MIME type
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		isBinary := !strings.HasPrefix(mimeType, "text/") &&
			!strings.Contains(mimeType, "application/json") &&
			!strings.Contains(mimeType, "application/xml")

		// If MIME type is definitive, cache and return
		if isBinary || strings.HasPrefix(mimeType, "text/") {
			if cache != nil {
				cache.Set(cacheKey, isBinary)
			}
			return isBinary
		}
	}

	// Read first bytes to detect null characters
	file, err := os.Open(filepath)
	if err != nil {
		if cache != nil {
			cache.Set(cacheKey, true) // When in doubt, consider as binary
		}
		return true
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		if cache != nil {
			cache.Set(cacheKey, true)
		}
		return true
	}

	isBinary := http.DetectContentType(buffer[:n]) != "text/plain; charset=utf-8"
	if cache != nil {
		cache.Set(cacheKey, isBinary)
	}
	return isBinary
} 