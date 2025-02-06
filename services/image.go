package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var (
	imageCache     = make(map[string]imageCacheEntry)
	imageCacheMux  sync.RWMutex
	cacheLifetime  = 24 * time.Hour
	unsplashClient = &http.Client{Timeout: 10 * time.Second}
)

type imageCacheEntry struct {
	URL       string
	CachedAt  time.Time
	Category  string
	Thumbnail string
}

type unsplashResponse struct {
	ID   string `json:"id"`
	URLs struct {
		Regular string `json:"regular"`
		Small   string `json:"small"`
		Thumb   string `json:"thumb"`
		Raw     string `json:"raw"`
	} `json:"urls"`
	Links struct {
		HTML string `json:"html"`
	} `json:"links"`
}

// FetchImage gets an image URL for a given category, using cache when possible
func FetchImage(category string) (string, error) {
	// Check cache first
	imageCacheMux.RLock()
	if entry, exists := imageCache[category]; exists && time.Since(entry.CachedAt) < cacheLifetime {
		imageCacheMux.RUnlock()
		return entry.URL, nil
	}
	imageCacheMux.RUnlock()

	// Fetch new image from Unsplash
	accessKey := os.Getenv("UNSPLASH_ACCESS_KEY")
	if accessKey == "" {
		return "", fmt.Errorf("UNSPLASH_ACCESS_KEY not set")
	}

	apiURL := fmt.Sprintf(
		"https://api.unsplash.com/photos/random?query=%s&orientation=landscape&client_id=%s",
		url.QueryEscape(category),
		accessKey,
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := unsplashClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unsplash API error: %s", resp.Status)
	}

	var result unsplashResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	// Cache the result
	imageCacheMux.Lock()
	imageCache[category] = imageCacheEntry{
		URL:       result.URLs.Regular,
		CachedAt:  time.Now(),
		Category:  category,
		Thumbnail: result.URLs.Thumb,
	}
	imageCacheMux.Unlock()

	// Log attribution (required by Unsplash)
	log.Printf("Photo by Unsplash photographer - View at: %s", result.Links.HTML)

	return result.URLs.Regular, nil
}

// ClearImageCache clears the image cache
func ClearImageCache() {
	imageCacheMux.Lock()
	imageCache = make(map[string]imageCacheEntry)
	imageCacheMux.Unlock()
}
