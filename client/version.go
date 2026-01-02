package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Version is set at build time via ldflags
var Version = "dev"

const (
	repoOwner      = "kuringer"
	repoName       = "alza-cli"
	checkInterval  = 24 * time.Hour
	cacheFileName  = "version_check.json"
)

type versionCache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate checks GitHub for newer version, returns message if update available
// Uses cache to avoid checking too frequently
func CheckForUpdate() string {
	cache, cacheFile := loadVersionCache()
	
	// Skip if checked recently
	if time.Since(cache.LastCheck) < checkInterval && cache.LatestVersion != "" {
		return formatUpdateMessage(cache.LatestVersion)
	}
	
	// Check GitHub (with short timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return ""
	}
	
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}
	
	// Save to cache
	cache.LastCheck = time.Now()
	cache.LatestVersion = release.TagName
	saveVersionCache(cacheFile, cache)
	
	return formatUpdateMessage(release.TagName)
}

func formatUpdateMessage(latestVersion string) string {
	if latestVersion == "" {
		return ""
	}
	
	// Normalize versions for comparison (remove 'v' prefix)
	current := strings.TrimPrefix(Version, "v")
	latest := strings.TrimPrefix(latestVersion, "v")
	
	if current == "dev" || current == latest {
		return ""
	}
	
	// Simple string comparison (works for semver)
	if latest > current {
		return fmt.Sprintf("\n💡 New version available: %s (you have %s)\n   Download: https://github.com/%s/%s/releases/latest\n",
			latestVersion, Version, repoOwner, repoName)
	}
	
	return ""
}

func loadVersionCache() (versionCache, string) {
	var cache versionCache
	
	configDir, err := ConfigDir()
	if err != nil {
		return cache, ""
	}
	
	cacheFile := filepath.Join(configDir, cacheFileName)
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return cache, cacheFile
	}
	
	json.Unmarshal(data, &cache)
	return cache, cacheFile
}

func saveVersionCache(cacheFile string, cache versionCache) {
	if cacheFile == "" {
		return
	}
	
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	
	os.WriteFile(cacheFile, data, 0644)
}
