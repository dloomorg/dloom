package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	cacheFileName = "update_check.json"
	cacheTTL      = 24 * time.Hour
	apiURL        = "https://api.github.com/repos/dloomorg/dloom/releases/latest"
	releasesURL   = "https://github.com/dloomorg/dloom/releases/latest"
)

type CacheEntry struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
}

func cachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dloom", cacheFileName), nil
}

func loadCache() (*CacheEntry, error) {
	path, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func saveCache(entry *CacheEntry) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// FetchLatestVersion calls the GitHub Releases API and returns the latest stable release tag.
func FetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.Prerelease {
		return "", nil
	}
	return release.TagName, nil
}

// CheckAndCache fetches the latest version if the cache is stale and saves the result.
// Safe to call from a goroutine; all errors are silently ignored.
func CheckAndCache() {
	if os.Getenv("DLOOM_NO_UPDATE_CHECK") != "" {
		return
	}
	cache, _ := loadCache()
	if cache != nil && time.Since(cache.CheckedAt) < cacheTTL {
		return
	}
	latest, err := FetchLatestVersion()
	if err != nil || latest == "" {
		return
	}
	_ = saveCache(&CacheEntry{CheckedAt: time.Now(), LatestVersion: latest})
}

// PendingNotice returns a non-empty notice string if a newer stable version is cached.
func PendingNotice(currentVersion string) string {
	if currentVersion == "" || currentVersion == "dev" {
		return ""
	}
	if os.Getenv("DLOOM_NO_UPDATE_CHECK") != "" {
		return ""
	}
	cache, err := loadCache()
	if err != nil || cache == nil {
		return ""
	}
	if isNewer(cache.LatestVersion, currentVersion) {
		return fmt.Sprintf(
			"A new version of dloom is available: %s (you have %s)\n       %s",
			cache.LatestVersion, currentVersion, releasesURL,
		)
	}
	return ""
}

// isNewer returns true if candidate is a strictly newer semver than current.
func isNewer(candidate, current string) bool {
	cv := parseSemver(candidate)
	cu := parseSemver(current)
	for i := range cv {
		if cv[i] > cu[i] {
			return true
		}
		if cv[i] < cu[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.Index(v, "-"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		result[i] = leadingInt(parts[i])
	}
	return result
}

func leadingInt(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	n, _ := strconv.Atoi(s[:i])
	return n
}
