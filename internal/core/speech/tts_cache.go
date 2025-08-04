package speech

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TTSCache provides intelligent caching for TTS audio chunks
type TTSCache struct {
	chunkCache      map[string]*CacheEntry
	phraseCache     map[string]*CacheEntry
	voiceModelCache map[string]*VoiceModel
	metricsTracker  *CacheMetrics
	config          *CacheConfig
	logger          *logrus.Logger
	mu              sync.RWMutex
	cleanupTicker   *time.Ticker
	stopChan        chan struct{}
}

// CacheEntry represents a cached audio item
type CacheEntry struct {
	Key          string                 `json:"key"`
	AudioData    []byte                 `json:"-"`          // Audio data (not serialized)
	AudioFile    string                 `json:"audio_file"` // Path to audio file
	Voice        string                 `json:"voice"`
	Language     string                 `json:"language"`
	Speed        float32                `json:"speed"`
	Duration     time.Duration          `json:"duration"`
	Size         int64                  `json:"size"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	AccessCount  int64                  `json:"access_count"`
	Metadata     map[string]interface{} `json:"metadata"`
	ExpiresAt    time.Time              `json:"expires_at"`
	IsPreloaded  bool                   `json:"is_preloaded"` // For common phrases
	Priority     int                    `json:"priority"`     // Higher = more important
	Hash         string                 `json:"hash"`         // Content hash for verification
}

// VoiceModel represents a cached voice model
type VoiceModel struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	LoadedAt  time.Time `json:"loaded_at"`
	LastUsed  time.Time `json:"last_used"`
	UseCount  int64     `json:"use_count"`
	IsDefault bool      `json:"is_default"`
	Language  string    `json:"language"`
	Gender    string    `json:"gender"`
	Quality   string    `json:"quality"`
}

// CacheMetrics tracks cache performance and usage
type CacheMetrics struct {
	TotalRequests    int64        `json:"total_requests"`
	CacheHits        int64        `json:"cache_hits"`
	CacheMisses      int64        `json:"cache_misses"`
	HitRate          float64      `json:"hit_rate"`
	TotalSize        int64        `json:"total_size"`
	EntryCount       int64        `json:"entry_count"`
	EvictionCount    int64        `json:"eviction_count"`
	LastCleanup      time.Time    `json:"last_cleanup"`
	AverageEntrySize int64        `json:"average_entry_size"`
	mu               sync.RWMutex `json:"-"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled              bool          `json:"enabled"`
	MaxCacheSize         int64         `json:"max_cache_size"`         // Total cache size in bytes
	MaxEntries           int           `json:"max_entries"`            // Maximum number of entries
	TTL                  time.Duration `json:"ttl"`                    // Time to live for entries
	CleanupInterval      time.Duration `json:"cleanup_interval"`       // How often to run cleanup
	PersistentCache      bool          `json:"persistent_cache"`       // Save cache to disk
	CacheDirectory       string        `json:"cache_directory"`        // Directory for cache files
	PreloadCommonPhrases bool          `json:"preload_common_phrases"` // Preload frequent phrases
	CompressionEnabled   bool          `json:"compression_enabled"`    // Compress cached audio
	EvictionPolicy       string        `json:"eviction_policy"`        // "lru", "lfu", "ttl"
	MaxFileSize          int64         `json:"max_file_size"`          // Max size of individual cached files
}

// Common phrases for preloading
var CommonPhrases = []string{
	"Hello", "Hi", "Good morning", "Good afternoon", "Good evening", "Good night",
	"Thank you", "You're welcome", "Please", "Excuse me", "I'm sorry",
	"Yes", "No", "Maybe", "Okay", "Alright", "Sure", "Certainly",
	"How are you?", "I'm fine", "What's your name?", "Nice to meet you",
	"Goodbye", "See you later", "Take care", "Have a good day",
	"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten",
}

// NewTTSCache creates a new TTS cache with default configuration
func NewTTSCache(logger *logrus.Logger) *TTSCache {
	config := &CacheConfig{
		Enabled:              true,
		MaxCacheSize:         500 * 1024 * 1024, // 500MB
		MaxEntries:           10000,
		TTL:                  24 * time.Hour,
		CleanupInterval:      time.Hour,
		PersistentCache:      true,
		CacheDirectory:       "./data/cache/tts",
		PreloadCommonPhrases: true,
		CompressionEnabled:   false, // Disabled for now to avoid complexity
		EvictionPolicy:       "lru",
		MaxFileSize:          10 * 1024 * 1024, // 10MB per file
	}

	cache := &TTSCache{
		chunkCache:      make(map[string]*CacheEntry),
		phraseCache:     make(map[string]*CacheEntry),
		voiceModelCache: make(map[string]*VoiceModel),
		metricsTracker: &CacheMetrics{
			LastCleanup: time.Now(),
		},
		config:   config,
		logger:   logger,
		stopChan: make(chan struct{}),
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(config.CacheDirectory, 0755); err != nil {
		logger.WithError(err).Error("Failed to create cache directory")
	}

	// Start cleanup routine
	cache.startCleanupRoutine()

	// Load persistent cache if enabled
	if config.PersistentCache {
		cache.loadPersistentCache()
	}

	// Preload common phrases if enabled
	if config.PreloadCommonPhrases {
		go cache.preloadCommonPhrases()
	}

	logger.Info("TTS cache initialized successfully")
	return cache
}

// GetCachedAudio retrieves cached audio for the given text and voice
func (tc *TTSCache) GetCachedAudio(text, voice string) ([]byte, bool) {
	if !tc.config.Enabled {
		return nil, false
	}

	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.TotalRequests++
	tc.metricsTracker.mu.Unlock()

	key := tc.generateKey(text, voice, "")

	tc.mu.RLock()
	entry, exists := tc.chunkCache[key]
	if !exists {
		// Try phrase cache
		entry, exists = tc.phraseCache[key]
	}
	tc.mu.RUnlock()

	if !exists {
		tc.recordCacheMiss()
		return nil, false
	}

	// Check if entry is expired
	if time.Now().After(entry.ExpiresAt) {
		tc.mu.Lock()
		delete(tc.chunkCache, key)
		delete(tc.phraseCache, key)
		tc.mu.Unlock()
		tc.recordCacheMiss()
		return nil, false
	}

	// Update access information
	tc.mu.Lock()
	entry.LastAccessed = time.Now()
	entry.AccessCount++
	tc.mu.Unlock()

	// Load audio data if not in memory
	var audioData []byte
	if len(entry.AudioData) > 0 {
		audioData = entry.AudioData
	} else if entry.AudioFile != "" {
		data, err := os.ReadFile(entry.AudioFile)
		if err != nil {
			tc.logger.WithError(err).WithField("file", entry.AudioFile).Error("Failed to read cached audio file")
			tc.recordCacheMiss()
			return nil, false
		}
		audioData = data
	} else {
		tc.recordCacheMiss()
		return nil, false
	}

	tc.recordCacheHit()
	tc.logger.WithFields(logrus.Fields{
		"key":          key,
		"size":         len(audioData),
		"access_count": entry.AccessCount,
	}).Debug("Cache hit for audio")

	return audioData, true
}

// CacheAudio stores audio data in the cache
func (tc *TTSCache) CacheAudio(text, voice string, audioData []byte) {
	if !tc.config.Enabled || len(audioData) == 0 {
		return
	}

	// Check size limits
	if int64(len(audioData)) > tc.config.MaxFileSize {
		tc.logger.WithFields(logrus.Fields{
			"size":     len(audioData),
			"max_size": tc.config.MaxFileSize,
		}).Warn("Audio too large to cache")
		return
	}

	key := tc.generateKey(text, voice, "")

	// Calculate content hash for verification
	hash := sha256.Sum256(audioData)
	hashString := hex.EncodeToString(hash[:])

	// Create cache entry
	entry := &CacheEntry{
		Key:          key,
		Voice:        voice,
		Duration:     tc.estimateAudioDuration(audioData),
		Size:         int64(len(audioData)),
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
		ExpiresAt:    time.Now().Add(tc.config.TTL),
		Priority:     tc.calculatePriority(text),
		Hash:         hashString,
		Metadata:     make(map[string]interface{}),
	}

	// Store to disk if persistent cache is enabled
	if tc.config.PersistentCache {
		filename := fmt.Sprintf("tts_%s.wav", hashString[:16])
		filePath := filepath.Join(tc.config.CacheDirectory, filename)

		if err := os.WriteFile(filePath, audioData, 0644); err != nil {
			tc.logger.WithError(err).Error("Failed to write cache file")
		} else {
			entry.AudioFile = filePath
		}
	} else {
		// Store in memory
		entry.AudioData = audioData
	}

	// Check if we need to evict entries before adding
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Calculate current cache size
	currentSize := tc.calculateCacheSize()

	// Evict entries if necessary
	if currentSize+entry.Size > tc.config.MaxCacheSize || len(tc.chunkCache) >= tc.config.MaxEntries {
		tc.evictEntries(entry.Size)
	}

	// Add to appropriate cache
	if tc.isCommonPhrase(text) {
		entry.IsPreloaded = true
		tc.phraseCache[key] = entry
	} else {
		tc.chunkCache[key] = entry
	}

	// Update metrics
	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.EntryCount++
	tc.metricsTracker.TotalSize += entry.Size
	tc.metricsTracker.mu.Unlock()

	tc.logger.WithFields(logrus.Fields{
		"key":       key,
		"size":      entry.Size,
		"cached_to": "phrase_cache",
	}).Debug("Audio cached successfully")
}

// PreloadCommonPhrases preloads frequently used phrases
func (tc *TTSCache) preloadCommonPhrases() {
	if !tc.config.PreloadCommonPhrases {
		return
	}

	tc.logger.Info("Starting preload of common phrases")

	// This would typically involve generating TTS for common phrases
	// For now, we'll create placeholder entries
	for _, phrase := range CommonPhrases {
		key := tc.generateKey(phrase, "default", "")

		// Check if already cached
		tc.mu.RLock()
		_, exists := tc.phraseCache[key]
		tc.mu.RUnlock()

		if !exists {
			// In a real implementation, you would generate TTS for the phrase here
			// For now, we'll create a placeholder
			tc.logger.WithField("phrase", phrase).Debug("Would preload phrase")
		}
	}
}

// CacheVoiceModel caches a voice model for faster loading
func (tc *TTSCache) CacheVoiceModel(name, path string) error {
	if !tc.config.Enabled {
		return nil
	}

	// Get file info
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat voice model file: %w", err)
	}

	model := &VoiceModel{
		Name:     name,
		Path:     path,
		Size:     fileInfo.Size(),
		LoadedAt: time.Now(),
		LastUsed: time.Now(),
		UseCount: 1,
	}

	tc.mu.Lock()
	tc.voiceModelCache[name] = model
	tc.mu.Unlock()

	tc.logger.WithFields(logrus.Fields{
		"name": name,
		"path": path,
		"size": model.Size,
	}).Info("Voice model cached")

	return nil
}

// GetCachedVoiceModel retrieves a cached voice model
func (tc *TTSCache) GetCachedVoiceModel(name string) (*VoiceModel, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	model, exists := tc.voiceModelCache[name]
	if exists {
		// Update usage info
		model.LastUsed = time.Now()
		model.UseCount++
	}

	return model, exists
}

// CleanupExpiredEntries removes expired cache entries
func (tc *TTSCache) CleanupExpiredEntries() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	removedCount := 0
	freedSize := int64(0)

	// Clean chunk cache
	for key, entry := range tc.chunkCache {
		if now.After(entry.ExpiresAt) {
			// Remove file if it exists
			if entry.AudioFile != "" {
				os.Remove(entry.AudioFile)
			}

			freedSize += entry.Size
			delete(tc.chunkCache, key)
			removedCount++
		}
	}

	// Clean phrase cache (but keep preloaded entries)
	for key, entry := range tc.phraseCache {
		if !entry.IsPreloaded && now.After(entry.ExpiresAt) {
			if entry.AudioFile != "" {
				os.Remove(entry.AudioFile)
			}

			freedSize += entry.Size
			delete(tc.phraseCache, key)
			removedCount++
		}
	}

	// Update metrics
	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.EntryCount -= int64(removedCount)
	tc.metricsTracker.TotalSize -= freedSize
	tc.metricsTracker.LastCleanup = now
	tc.metricsTracker.mu.Unlock()

	if removedCount > 0 {
		tc.logger.WithFields(logrus.Fields{
			"removed_count": removedCount,
			"freed_size":    freedSize,
		}).Info("Cache cleanup completed")
	}
}

// GetCacheStats returns current cache statistics
func (tc *TTSCache) GetCacheStats() map[string]interface{} {
	tc.metricsTracker.mu.RLock()
	defer tc.metricsTracker.mu.RUnlock()

	tc.mu.RLock()
	chunkCount := len(tc.chunkCache)
	phraseCount := len(tc.phraseCache)
	voiceModelCount := len(tc.voiceModelCache)
	tc.mu.RUnlock()

	// Calculate hit rate
	hitRate := 0.0
	if tc.metricsTracker.TotalRequests > 0 {
		hitRate = float64(tc.metricsTracker.CacheHits) / float64(tc.metricsTracker.TotalRequests)
	}

	// Calculate average entry size
	avgSize := int64(0)
	if tc.metricsTracker.EntryCount > 0 {
		avgSize = tc.metricsTracker.TotalSize / tc.metricsTracker.EntryCount
	}

	return map[string]interface{}{
		"enabled":            tc.config.Enabled,
		"total_requests":     tc.metricsTracker.TotalRequests,
		"cache_hits":         tc.metricsTracker.CacheHits,
		"cache_misses":       tc.metricsTracker.CacheMisses,
		"hit_rate":           hitRate,
		"total_size":         tc.metricsTracker.TotalSize,
		"max_cache_size":     tc.config.MaxCacheSize,
		"entry_count":        tc.metricsTracker.EntryCount,
		"max_entries":        tc.config.MaxEntries,
		"chunk_cache_count":  chunkCount,
		"phrase_cache_count": phraseCount,
		"voice_model_count":  voiceModelCount,
		"eviction_count":     tc.metricsTracker.EvictionCount,
		"last_cleanup":       tc.metricsTracker.LastCleanup,
		"average_entry_size": avgSize,
		"cache_directory":    tc.config.CacheDirectory,
		"ttl":                tc.config.TTL.String(),
	}
}

// Clear removes all entries from the cache
func (tc *TTSCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Remove all files
	for _, entry := range tc.chunkCache {
		if entry.AudioFile != "" {
			os.Remove(entry.AudioFile)
		}
	}
	for _, entry := range tc.phraseCache {
		if entry.AudioFile != "" {
			os.Remove(entry.AudioFile)
		}
	}

	// Clear maps
	tc.chunkCache = make(map[string]*CacheEntry)
	tc.phraseCache = make(map[string]*CacheEntry)
	tc.voiceModelCache = make(map[string]*VoiceModel)

	// Reset metrics
	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.EntryCount = 0
	tc.metricsTracker.TotalSize = 0
	tc.metricsTracker.mu.Unlock()

	tc.logger.Info("Cache cleared")
}

// Stop gracefully shuts down the cache
func (tc *TTSCache) Stop() {
	close(tc.stopChan)
	if tc.cleanupTicker != nil {
		tc.cleanupTicker.Stop()
	}

	// Save persistent cache if enabled
	if tc.config.PersistentCache {
		tc.savePersistentCache()
	}

	tc.logger.Info("TTS cache stopped")
}

// Private methods

func (tc *TTSCache) generateKey(text, voice, language string) string {
	// Create a hash of the input parameters
	input := fmt.Sprintf("%s|%s|%s", text, voice, language)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func (tc *TTSCache) estimateAudioDuration(audioData []byte) time.Duration {
	// Simple estimation based on file size
	// This is a rough approximation for WAV files
	// Real implementation would parse the audio header
	if len(audioData) < 44 {
		return 0
	}

	// Assume 16-bit, 22050 Hz, mono
	samplesPerSecond := 22050
	bytesPerSample := 2
	dataSize := len(audioData) - 44 // Subtract WAV header size

	durationSeconds := float64(dataSize) / float64(samplesPerSecond*bytesPerSample)
	return time.Duration(durationSeconds * float64(time.Second))
}

func (tc *TTSCache) calculatePriority(text string) int {
	priority := 5 // Default priority

	if tc.isCommonPhrase(text) {
		priority = 10 // High priority for common phrases
	} else if len(text) < 50 {
		priority = 8 // Higher priority for short text
	} else if len(text) > 500 {
		priority = 3 // Lower priority for long text
	}

	return priority
}

func (tc *TTSCache) isCommonPhrase(text string) bool {
	for _, phrase := range CommonPhrases {
		if text == phrase {
			return true
		}
	}
	return false
}

func (tc *TTSCache) calculateCacheSize() int64 {
	size := int64(0)
	for _, entry := range tc.chunkCache {
		size += entry.Size
	}
	for _, entry := range tc.phraseCache {
		size += entry.Size
	}
	return size
}

func (tc *TTSCache) evictEntries(neededSpace int64) {
	switch tc.config.EvictionPolicy {
	case "lfu":
		tc.evictLFU(neededSpace)
	case "ttl":
		tc.evictTTL(neededSpace)
	default:
		tc.evictLRU(neededSpace)
	}
}

func (tc *TTSCache) evictLRU(neededSpace int64) {
	// Collect all entries and sort by last accessed time
	var entries []*CacheEntry
	var keys []string

	for key, entry := range tc.chunkCache {
		if !entry.IsPreloaded {
			entries = append(entries, entry)
			keys = append(keys, key)
		}
	}

	// Sort by last accessed (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastAccessed.Before(entries[j].LastAccessed)
	})

	freedSpace := int64(0)
	evictedCount := 0

	for i, entry := range entries {
		if freedSpace >= neededSpace && len(tc.chunkCache) < tc.config.MaxEntries {
			break
		}

		key := keys[i]

		// Remove file if it exists
		if entry.AudioFile != "" {
			os.Remove(entry.AudioFile)
		}

		freedSpace += entry.Size
		delete(tc.chunkCache, key)
		evictedCount++
	}

	// Update metrics
	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.EvictionCount += int64(evictedCount)
	tc.metricsTracker.EntryCount -= int64(evictedCount)
	tc.metricsTracker.TotalSize -= freedSpace
	tc.metricsTracker.mu.Unlock()

	if evictedCount > 0 {
		tc.logger.WithFields(logrus.Fields{
			"evicted_count": evictedCount,
			"freed_space":   freedSpace,
			"policy":        "LRU",
		}).Info("Cache entries evicted")
	}
}

func (tc *TTSCache) evictLFU(neededSpace int64) {
	// Similar to LRU but sort by access count
	var entries []*CacheEntry
	var keys []string

	for key, entry := range tc.chunkCache {
		if !entry.IsPreloaded {
			entries = append(entries, entry)
			keys = append(keys, key)
		}
	}

	// Sort by access count (least used first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AccessCount < entries[j].AccessCount
	})

	freedSpace := int64(0)
	evictedCount := 0

	for i, entry := range entries {
		if freedSpace >= neededSpace && len(tc.chunkCache) < tc.config.MaxEntries {
			break
		}

		key := keys[i]

		if entry.AudioFile != "" {
			os.Remove(entry.AudioFile)
		}

		freedSpace += entry.Size
		delete(tc.chunkCache, key)
		evictedCount++
	}

	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.EvictionCount += int64(evictedCount)
	tc.metricsTracker.EntryCount -= int64(evictedCount)
	tc.metricsTracker.TotalSize -= freedSpace
	tc.metricsTracker.mu.Unlock()

	if evictedCount > 0 {
		tc.logger.WithFields(logrus.Fields{
			"evicted_count": evictedCount,
			"freed_space":   freedSpace,
			"policy":        "LFU",
		}).Info("Cache entries evicted")
	}
}

func (tc *TTSCache) evictTTL(neededSpace int64) {
	// Evict entries that are closest to expiring
	var entries []*CacheEntry
	var keys []string

	for key, entry := range tc.chunkCache {
		if !entry.IsPreloaded {
			entries = append(entries, entry)
			keys = append(keys, key)
		}
	}

	// Sort by expiration time (earliest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ExpiresAt.Before(entries[j].ExpiresAt)
	})

	freedSpace := int64(0)
	evictedCount := 0

	for i, entry := range entries {
		if freedSpace >= neededSpace && len(tc.chunkCache) < tc.config.MaxEntries {
			break
		}

		key := keys[i]

		if entry.AudioFile != "" {
			os.Remove(entry.AudioFile)
		}

		freedSpace += entry.Size
		delete(tc.chunkCache, key)
		evictedCount++
	}

	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.EvictionCount += int64(evictedCount)
	tc.metricsTracker.EntryCount -= int64(evictedCount)
	tc.metricsTracker.TotalSize -= freedSpace
	tc.metricsTracker.mu.Unlock()

	if evictedCount > 0 {
		tc.logger.WithFields(logrus.Fields{
			"evicted_count": evictedCount,
			"freed_space":   freedSpace,
			"policy":        "TTL",
		}).Info("Cache entries evicted")
	}
}

func (tc *TTSCache) recordCacheHit() {
	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.CacheHits++
	tc.metricsTracker.mu.Unlock()
}

func (tc *TTSCache) recordCacheMiss() {
	tc.metricsTracker.mu.Lock()
	tc.metricsTracker.CacheMisses++
	tc.metricsTracker.mu.Unlock()
}

func (tc *TTSCache) startCleanupRoutine() {
	tc.cleanupTicker = time.NewTicker(tc.config.CleanupInterval)

	go func() {
		for {
			select {
			case <-tc.cleanupTicker.C:
				tc.CleanupExpiredEntries()
			case <-tc.stopChan:
				return
			}
		}
	}()
}

func (tc *TTSCache) savePersistentCache() {
	// Save cache metadata to a JSON file
	metadataPath := filepath.Join(tc.config.CacheDirectory, "cache_metadata.json")

	metadata := struct {
		ChunkCache  map[string]*CacheEntry `json:"chunk_cache"`
		PhraseCache map[string]*CacheEntry `json:"phrase_cache"`
		SavedAt     time.Time              `json:"saved_at"`
	}{
		ChunkCache:  tc.chunkCache,
		PhraseCache: tc.phraseCache,
		SavedAt:     time.Now(),
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		tc.logger.WithError(err).Error("Failed to marshal cache metadata")
		return
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		tc.logger.WithError(err).Error("Failed to save cache metadata")
		return
	}

	tc.logger.Info("Cache metadata saved successfully")
}

func (tc *TTSCache) loadPersistentCache() {
	metadataPath := filepath.Join(tc.config.CacheDirectory, "cache_metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			tc.logger.WithError(err).Error("Failed to read cache metadata")
		}
		return
	}

	var metadata struct {
		ChunkCache  map[string]*CacheEntry `json:"chunk_cache"`
		PhraseCache map[string]*CacheEntry `json:"phrase_cache"`
		SavedAt     time.Time              `json:"saved_at"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		tc.logger.WithError(err).Error("Failed to unmarshal cache metadata")
		return
	}

	// Load and validate entries
	now := time.Now()
	loadedCount := 0

	for key, entry := range metadata.ChunkCache {
		// Skip expired entries
		if now.After(entry.ExpiresAt) {
			continue
		}

		// Verify file exists
		if entry.AudioFile != "" {
			if _, err := os.Stat(entry.AudioFile); os.IsNotExist(err) {
				continue
			}
		}

		tc.chunkCache[key] = entry
		loadedCount++
	}

	for key, entry := range metadata.PhraseCache {
		if now.After(entry.ExpiresAt) && !entry.IsPreloaded {
			continue
		}

		if entry.AudioFile != "" {
			if _, err := os.Stat(entry.AudioFile); os.IsNotExist(err) {
				continue
			}
		}

		tc.phraseCache[key] = entry
		loadedCount++
	}

	tc.logger.WithFields(logrus.Fields{
		"loaded_count": loadedCount,
		"saved_at":     metadata.SavedAt,
	}).Info("Persistent cache loaded successfully")
}
