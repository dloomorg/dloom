package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// useTempCache points the user cache dir at a temp dir and clears the opt-out env var.
// It returns the cache file path.
func useTempCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv(disableEnvVar, "")
	path, err := cachePath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// fakeAPI serves the given status and body in place of the GitHub API and counts requests.
func fakeAPI(t *testing.T, status int, body string) *atomic.Int32 {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	original := apiURL
	apiURL = server.URL
	t.Cleanup(func() { apiURL = original })
	return &hits
}

func writeCache(t *testing.T, checkedAt time.Time, latest string) {
	t.Helper()
	if err := saveCache(&CacheEntry{CheckedAt: checkedAt, LatestVersion: latest}); err != nil {
		t.Fatal(err)
	}
}

func readCache(t *testing.T) *CacheEntry {
	t.Helper()
	entry, err := loadCache()
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("update check did not finish")
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		in   string
		want [3]int
	}{
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.2.3", [3]int{1, 2, 3}},
		{"v1.10.0", [3]int{1, 10, 0}},
		{"v1.0.3.rc.1", [3]int{1, 0, 3}},
		{"v1.2.3-beta.1", [3]int{1, 2, 3}},
		{"1.2", [3]int{1, 2, 0}},
		{"dev", [3]int{0, 0, 0}},
		{"", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseSemver(tt.in); got != tt.want {
				t.Errorf("parseSemver(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"v1.0.2", "1.0.0", true},
		{"v1.0.2", "v1.0.1", true},
		{"v1.1.0", "v1.0.9", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.10.0", "v1.9.0", true},
		{"v1.0.2", "1.0.2", false},
		{"v1.0.2", "v1.0.2", false},
		{"v1.0.1", "v1.0.2", false},
		{"v1.0.2", "v1.0.3.rc.1", false},
		{"", "v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.candidate+"_vs_"+tt.current, func(t *testing.T) {
			if got := isNewer(tt.candidate, tt.current); got != tt.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.candidate, tt.current, got, tt.want)
			}
		})
	}
}

func TestFetchLatestVersion(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{"stable release", http.StatusOK, `{"tag_name":"v1.0.2","prerelease":false}`, "v1.0.2", false},
		{"prerelease is ignored", http.StatusOK, `{"tag_name":"v1.0.3.rc.1","prerelease":true}`, "", false},
		{"non-200 status", http.StatusForbidden, `{"message":"rate limited"}`, "", true},
		{"invalid json", http.StatusOK, `not json`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeAPI(t, tt.status, tt.body)
			got, err := FetchLatestVersion()
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("FetchLatestVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStartCheck(t *testing.T) {
	const latestBody = `{"tag_name":"v1.0.2","prerelease":false}`
	fresh := time.Now().Add(-time.Hour)
	stale := time.Now().Add(-2 * cacheTTL)

	t.Run("fresh cache skips the network", func(t *testing.T) {
		useTempCache(t)
		hits := fakeAPI(t, http.StatusOK, latestBody)
		writeCache(t, fresh, "v1.0.1")

		waitDone(t, StartCheck())

		if hits.Load() != 0 {
			t.Errorf("expected no requests, got %d", hits.Load())
		}
		if got := readCache(t).LatestVersion; got != "v1.0.1" {
			t.Errorf("cached version = %q, want v1.0.1", got)
		}
	})

	t.Run("stale cache is refreshed", func(t *testing.T) {
		useTempCache(t)
		hits := fakeAPI(t, http.StatusOK, latestBody)
		writeCache(t, stale, "v1.0.1")

		waitDone(t, StartCheck())

		if hits.Load() != 1 {
			t.Errorf("expected 1 request, got %d", hits.Load())
		}
		entry := readCache(t)
		if entry.LatestVersion != "v1.0.2" {
			t.Errorf("cached version = %q, want v1.0.2", entry.LatestVersion)
		}
		if time.Since(entry.CheckedAt) > time.Minute {
			t.Errorf("checked_at not updated: %v", entry.CheckedAt)
		}
	})

	t.Run("missing cache is created", func(t *testing.T) {
		useTempCache(t)
		fakeAPI(t, http.StatusOK, latestBody)

		waitDone(t, StartCheck())

		if got := readCache(t).LatestVersion; got != "v1.0.2" {
			t.Errorf("cached version = %q, want v1.0.2", got)
		}
	})

	t.Run("corrupt cache is replaced", func(t *testing.T) {
		path := useTempCache(t)
		fakeAPI(t, http.StatusOK, latestBody)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{truncated"), 0644); err != nil {
			t.Fatal(err)
		}

		waitDone(t, StartCheck())

		if got := readCache(t).LatestVersion; got != "v1.0.2" {
			t.Errorf("cached version = %q, want v1.0.2", got)
		}
	})

	t.Run("failed check keeps previous version and resets the timer", func(t *testing.T) {
		useTempCache(t)
		fakeAPI(t, http.StatusInternalServerError, "")
		writeCache(t, stale, "v1.0.1")

		waitDone(t, StartCheck())

		entry := readCache(t)
		if entry.LatestVersion != "v1.0.1" {
			t.Errorf("cached version = %q, want v1.0.1", entry.LatestVersion)
		}
		if time.Since(entry.CheckedAt) > time.Minute {
			t.Errorf("checked_at not updated after failure: %v", entry.CheckedAt)
		}
	})

	t.Run("opt-out skips the check", func(t *testing.T) {
		path := useTempCache(t)
		t.Setenv(disableEnvVar, "1")
		hits := fakeAPI(t, http.StatusOK, latestBody)

		waitDone(t, StartCheck())

		if hits.Load() != 0 {
			t.Errorf("expected no requests, got %d", hits.Load())
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected no cache file, stat err = %v", err)
		}
	})
}

func TestPendingNotice(t *testing.T) {
	tests := []struct {
		name    string
		current string
		cached  string
		disable bool
		want    bool
	}{
		{"newer version cached", "1.0.0", "v1.0.2", false, true},
		{"same version cached", "1.0.2", "v1.0.2", false, false},
		{"older version cached", "1.0.3", "v1.0.2", false, false},
		{"failed check cached", "1.0.0", "", false, false},
		{"dev build", "dev", "v1.0.2", false, false},
		{"unset version", "", "v1.0.2", false, false},
		{"opted out", "1.0.0", "v1.0.2", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTempCache(t)
			writeCache(t, time.Now(), tt.cached)
			if tt.disable {
				t.Setenv(disableEnvVar, "1")
			}

			notice := PendingNotice(tt.current)
			if got := notice != ""; got != tt.want {
				t.Fatalf("PendingNotice(%q) = %q, want notice: %v", tt.current, notice, tt.want)
			}
			if tt.want {
				for _, s := range []string{tt.cached, tt.current, releasesURL} {
					if !strings.Contains(notice, s) {
						t.Errorf("notice %q does not mention %q", notice, s)
					}
				}
			}
		})
	}

	t.Run("no cache", func(t *testing.T) {
		useTempCache(t)
		if notice := PendingNotice("1.0.0"); notice != "" {
			t.Errorf("PendingNotice() = %q, want empty", notice)
		}
	})
}

func TestWait(t *testing.T) {
	t.Run("nil channel returns immediately", func(t *testing.T) {
		start := time.Now()
		Wait(nil, time.Second)
		if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
			t.Errorf("Wait(nil) took %v", elapsed)
		}
	})

	t.Run("closed channel returns immediately", func(t *testing.T) {
		done := make(chan struct{})
		close(done)
		start := time.Now()
		Wait(done, time.Second)
		if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
			t.Errorf("Wait(closed) took %v", elapsed)
		}
	})

	t.Run("open channel returns after timeout", func(t *testing.T) {
		start := time.Now()
		Wait(make(chan struct{}), 20*time.Millisecond)
		if elapsed := time.Since(start); elapsed < 20*time.Millisecond || elapsed > time.Second {
			t.Errorf("Wait(open, 20ms) took %v", elapsed)
		}
	})
}

func TestSaveCacheLeavesNoTempFiles(t *testing.T) {
	path := useTempCache(t)
	writeCache(t, time.Now(), "v1.0.1")
	writeCache(t, time.Now(), "v1.0.2")

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != cacheFileName {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("cache dir contains %v, want only %s", names, cacheFileName)
	}
	if got := readCache(t).LatestVersion; got != "v1.0.2" {
		t.Errorf("cached version = %q, want v1.0.2", got)
	}
}
