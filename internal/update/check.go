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
	disableEnvVar = "DLOOM_NO_UPDATE_CHECK"
	releasesURL   = "https://github.com/dloomorg/dloom/releases/latest"
)

// apiURL is a variable so tests can point it at a local server.
var apiURL = "https://api.github.com/repos/dloomorg/dloom/releases/latest"

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
	// Write to a temp file and rename so an interrupted write never leaves a partial cache.
	tmp, err := os.CreateTemp(filepath.Dir(path), cacheFileName+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
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

// StartCheck refreshes the cached latest version in the background if the cache is stale.
// The returned channel is closed when the check finishes, or immediately if no check is needed.
func StartCheck() <-chan struct{} {
	done := make(chan struct{})
	if os.Getenv(disableEnvVar) != "" {
		close(done)
		return done
	}
	cache, _ := loadCache()
	if cache != nil && time.Since(cache.CheckedAt) < cacheTTL {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		refreshCache(cache)
	}()
	return done
}

// Wait blocks until done is closed or timeout elapses, whichever comes first.
func Wait(done <-chan struct{}, timeout time.Duration) {
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// refreshCache fetches the latest version and saves it. Failed fetches are cached too,
// keeping the previously known version, so offline or rate-limited users don't retry on every run.
func refreshCache(previous *CacheEntry) {
	entry := CacheEntry{CheckedAt: time.Now()}
	if previous != nil {
		entry.LatestVersion = previous.LatestVersion
	}
	if latest, err := FetchLatestVersion(); err == nil && latest != "" {
		entry.LatestVersion = latest
	}
	_ = saveCache(&entry)
}

// PendingNotice returns a non-empty notice string if a newer stable version is cached.
func PendingNotice(currentVersion string) string {
	if currentVersion == "" || currentVersion == "dev" {
		return ""
	}
	if os.Getenv(disableEnvVar) != "" {
		return ""
	}
	cache, err := loadCache()
	if err != nil || cache == nil {
		return ""
	}
	if isNewer(cache.LatestVersion, currentVersion) {
		return fmt.Sprintf(
			"A new version of dloom is available: %s (you have %s). Upgrade with your package manager or download it from %s",
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
