package proxy

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type Backend interface {
	GetResource(path string, w http.ResponseWriter, r *http.Request)
}

type BackendConfig struct {
	Prefix        string `yaml:"prefix"`
	Type          string `yaml:"repo_type"`
	BaseUrl       string `yaml:"url"`
	CacheDuration string `yaml:"cache_duration"`
}

type resourceStat struct {
	etag         string
	lastModified time.Time
}

type backendStatCache struct {
	lock  sync.RWMutex
	store map[string]resourceStat
}

func newBackendStatCache() *backendStatCache {
	return &backendStatCache{
		store: make(map[string]resourceStat),
	}
}

func (b *backendStatCache) set(name string, stat resourceStat) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.store[name] = stat
}

func (b *backendStatCache) get(name string) *resourceStat {
	b.lock.RLock()
	defer b.lock.RUnlock()
	res, ok := b.store[name]
	if ok {
		return &res
	} else {
		return nil
	}
}

type MvnBackend struct {
	BackendConfig
	backendStatCache
}

func (m *MvnBackend) GetResource(path string, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Allow", "GET,HEAD")
		return
	}
	stats := m.backendStatCache.get(path)
	// We have seen this resource before
	if stats != nil {
		// Do the conditional get dance
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.EqualFold(stats.etag, match) {
				w.WriteHeader(http.StatusNotModified)
				w.Header().Set("ETag", stats.etag)
				w.Header().Set("Last-Modified", stats.lastModified.Format(time.RFC1123))
				return
			}
		}
		if modifiedSince := r.Header.Get("If-Modified-Since"); modifiedSince != "" {
			expected, err := time.Parse(time.RFC1123, modifiedSince)
			if err == nil {
				if expected.After(stats.lastModified) || expected.Equal(stats.lastModified) {
					w.WriteHeader(http.StatusNotModified)
					w.Header().Set("ETag", stats.etag)
					w.Header().Set("Last-Modified", stats.lastModified.Format(time.RFC1123))
					return
				}
			}
		}
	}

}
