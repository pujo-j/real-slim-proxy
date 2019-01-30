package proxy

import (
	"encoding/base64"
	log "github.com/sirupsen/logrus"
	"gocloud.dev/blob"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Backend interface {
	GetResource(path string, w http.ResponseWriter, r *http.Request, bucket *blob.Bucket)
	GetConfig() BackendConfig
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
	*backendStatCache
	loadingCache      map[string]sync.Once
	loadingCacheMutex sync.Mutex
	client            *http.Client
}

func NewMvnBackend(config BackendConfig) *MvnBackend {
	return &MvnBackend{
		BackendConfig:    config,
		backendStatCache: newBackendStatCache(),
		loadingCache:     make(map[string]sync.Once),
		client:           http.DefaultClient,
	}
}

func (m *MvnBackend) sendResourceFromStorage(path string, w http.ResponseWriter, r *http.Request, bucket *blob.Bucket) bool {
	if r.Method != "GET" && r.Method != "HEAD" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Allow", "GET,HEAD")
		return true
	}
	stats := m.backendStatCache.get(path)
	// We have seen this resource before
	if stats != nil {
		// Do the conditional get dance
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.EqualFold(stats.etag, match) {
				w.WriteHeader(http.StatusNotModified)
				w.Header().Set("Accept-Ranges", "none")
				w.Header().Set("ETag", stats.etag)
				w.Header().Set("Last-Modified", stats.lastModified.Format(time.RFC1123))
				return true
			}
		}
		if modifiedSince := r.Header.Get("If-Modified-Since"); modifiedSince != "" {
			expected, err := time.Parse(time.RFC1123, modifiedSince)
			if err == nil {
				if expected.After(stats.lastModified) || expected.Equal(stats.lastModified) {
					w.WriteHeader(http.StatusNotModified)
					w.Header().Set("Accept-Ranges", "none")
					w.Header().Set("ETag", stats.etag)
					w.Header().Set("Last-Modified", stats.lastModified.Format(time.RFC1123))
					return true
				}
			}
		}
	}
	objectPath := m.BackendConfig.Prefix + "/" + path
	objectAttrs, err := bucket.Attributes(r.Context(), objectPath)
	if blob.IsNotExist(err) {
		return false
	}
	objectStats := resourceStat{base64.RawURLEncoding.EncodeToString(objectAttrs.MD5), objectAttrs.ModTime}
	if stats == nil || stats.lastModified.Before(objectAttrs.ModTime) {
		m.backendStatCache.set(path, objectStats)
	}
	if r.Method == "HEAD" {
		w.WriteHeader(200)
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("ETag", stats.etag)
		w.Header().Set("Last-Modified", stats.lastModified.Format(time.RFC1123))
		//TODO: Handle files > 2Go ?
		w.Header().Set("Content-Size", strconv.Itoa(int(objectAttrs.Size)))
	} else {
		object, err := bucket.NewReader(r.Context(), objectPath, nil)
		if blob.IsNotExist(err) {
			// Ok, we had weird race here, whatever
			w.WriteHeader(404)
			return false
		}
		w.WriteHeader(200)
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("ETag", stats.etag)
		w.Header().Set("Last-Modified", stats.lastModified.Format(time.RFC1123))
		//TODO: Handle files > 2Go ?
		w.Header().Set("Content-Size", strconv.Itoa(int(objectAttrs.Size)))
		_, err = io.Copy(w, object)
		log.WithError(err).Debug("sending object")
	}
	return true
}

func (m *MvnBackend) GetConfig() BackendConfig {
	return m.BackendConfig
}
func (m *MvnBackend) GetResource(path string, w http.ResponseWriter, r *http.Request, bucket *blob.Bucket) {
	if !m.sendResourceFromStorage(path, w, r, bucket) {
		m.loadingCacheMutex.Lock()
		loader, ok := m.loadingCache[path]
		if !ok {
			loader = sync.Once{}
			m.loadingCache[path] = loader
		}
		m.loadingCacheMutex.Unlock()
		loader.Do(func() {
			url := m.BaseUrl + path
			response, err := m.client.Get(url)
			if err != nil {
				log.WithError(err).Error("calling backend", m.Prefix)
				return
			}
			if response.StatusCode == http.StatusNotFound {
				log.WithField("url", url).WithField("response", response).Debug("backend resource not found")
				return
			}
			if response.StatusCode != http.StatusOK {
				log.WithField("response", response).Error("backend error")
			}
			writer, err := bucket.NewWriter(r.Context(), m.Prefix+"/"+path, &blob.WriterOptions{Metadata: map[string]string{"ETag": response.Header.Get("ETag")}})
			if err != nil {
				log.WithError(err).Error("writing in blob storage")
			}
			_, err = io.Copy(writer, response.Body)
			if err != nil {
				log.WithError(err).Error("writing in blob storage")
			}
			err = writer.Close()
			if err != nil {
				log.WithError(err).Error("writing in blob storage")
			}
			log.WithField("writer", writer).Debug("wrote to blob store")
			err = response.Body.Close()
			if err != nil {
				log.WithError(err).Error("closing client")
			}
		})
		foundNow := m.sendResourceFromStorage(path, w, r, bucket)
		if !foundNow {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
