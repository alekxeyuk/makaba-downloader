package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// dirCacheEntry stores cached directory scan results with metadata
type dirCacheEntry struct {
	files    map[string]struct{}
	modTime  time.Time
	cachedAt time.Time
}

// dirCache is a thread-safe cache for directory scan results
type dirCache struct {
	mu    sync.RWMutex
	cache map[string]*dirCacheEntry
}

// global directory cache instance
var globalDirCache = &dirCache{
	cache: make(map[string]*dirCacheEntry),
}

// get retrieves cached results if available and valid
func (dc *dirCache) get(dirName string) (map[string]struct{}, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	entry, exists := dc.cache[dirName]
	if !exists {
		return nil, false
	}

	// Check if directory still exists and get current mod time
	fi, err := os.Stat(dirName)
	if err != nil {
		// Directory no longer exists or is inaccessible, invalidate cache
		return nil, false
	}

	// If directory modification time hasn't changed, return cached result
	if fi.ModTime().Equal(entry.modTime) {
		return entry.files, true
	}

	return nil, false
}

// set stores scan results in the cache
func (dc *dirCache) set(dirName string, files map[string]struct{}, modTime time.Time) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.cache[dirName] = &dirCacheEntry{
		files:    files,
		modTime:  modTime,
		cachedAt: time.Now(),
	}
}

// invalidate removes a directory from the cache
func (dc *dirCache) invalidate(dirName string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	delete(dc.cache, dirName)
}

// clear removes all entries from the cache
func (dc *dirCache) clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.cache = make(map[string]*dirCacheEntry)
}

func loadLastHits() map[string]int64 {
	lastHits := make(map[string]int64)
	file, err := os.Open("lasthits.json")
	if err != nil {
		if !os.IsNotExist(err) {
			Log.Error("Error opening lasthits.json: %v", err)
		}
		return lastHits
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&lastHits)
	if err != nil {
		Log.Error("Error decoding lasthits.json: %v", err)
	}
	return lastHits
}

func saveLastHits(lastHits map[string]int64) {
	file, err := os.Create("lasthits.json")
	if err != nil {
		Log.Error("Error creating lasthits.json: %v", err)
		return
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(lastHits)
	if err != nil {
		Log.Error("Error encoding lasthits.json: %v", err)
	}
}

func getAlreadyHaveFiles(dirName string) map[string]struct{} {
	// Try to get cached results first
	if cachedFiles, found := globalDirCache.get(dirName); found {
		Log.Debug("Using cached results for directory: %s", dirName)
		return cachedFiles
	}

	// Cache miss - need to scan the directory
	Log.Debug("Cache miss - scanning directory: %s", dirName)
	alreadyHaveFiles := make(map[string]struct{})

	// Check if directory exists
	fi, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		Log.Info("Directory %s does not exist, skipping...", dirName)
		// Invalidate any stale cache entry for non-existent directory
		globalDirCache.invalidate(dirName)
		return alreadyHaveFiles
	}
	if err != nil {
		Log.Error("Error checking directory %s: %v", dirName, err)
		return alreadyHaveFiles
	}
	if !fi.IsDir() {
		Log.Error("Path %s is not a directory", dirName)
		return alreadyHaveFiles
	}

	// Store the directory's modification time for cache validation
	dirModTime := fi.ModTime()

	// Use WalkDir for better performance (available in Go 1.16+)
	err = filepath.WalkDir(dirName, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			name := info.Name()
			if len(name) >= 32 {
				alreadyHaveFiles[name[:32]] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		Log.Error("Error walking directory %s: %v", dirName, err)
		return alreadyHaveFiles
	}

	// Cache the results with the directory's modification time
	globalDirCache.set(dirName, alreadyHaveFiles, dirModTime)
	Log.Debug("Cached %d files from directory: %s", len(alreadyHaveFiles), dirName)

	return alreadyHaveFiles
}

func sanitizeFileName(fileName string) string {
	// Characters that are invalid in filenames on Windows and most Unix systems
	invalidChars := []string{":", "*", "?", "<", ">", "|", "\""}
	for _, char := range invalidChars {
		fileName = strings.ReplaceAll(fileName, char, "")
	}
	return fileName
}

func isValidFileExtension(file []byte, fileExtensions []string) bool {
	path := gjson.GetBytes(file, "path").String()
	ext := filepath.Ext(path)
	if len(ext) == 0 {
		return false
	}
	for _, allowedExt := range fileExtensions {
		if strings.EqualFold(ext[1:], allowedExt) {
			return true
		}
	}
	// Log unknown extension to unknown.txt
	unknownFile, err := os.OpenFile("unknown.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer unknownFile.Close()
		unknownFile.WriteString(ext[1:] + "\n")
	}
	return false
}
