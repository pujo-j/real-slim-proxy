package proxy

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocloud.dev/blob"
	"io"
	"net/http"
)

type Proxy struct {
	Server   *http.Server
	Backends map[string]Backend
	bucket   *blob.Bucket
}

type ServerConfig struct {
	Port     int                      `yaml:"listen_port"`
	Iface    string                   `yaml:"listen_address"`
	Store    StoreConfig              `yaml:"store"`
	Backends map[string]BackendConfig `yaml:"backends"`
}

func (s *ServerConfig) applyDefaults() {
	if s.Port == 0 {
		s.Port = 8678
	}
	if s.Store.Cloud == "" {
		s.Store.Cloud = "gcp"
	}
	for name, backend := range s.Backends {
		if backend.Type == "" {
			backend.Type = "mvn"
		}
		s.Backends[name] = backend
	}
}

func NewServer(config ServerConfig) (*Proxy, error) {
	config.applyDefaults()
	ctx := context.Background()
	bucket, err := config.Store.OpenStore(ctx)
	if err != nil {
		return nil, err
	}
	log.WithField("bucket", bucket).Debug("Created blob storage")
	li := bucket.List(&blob.ListOptions{Delimiter: "/"})
	_, err = li.Next(ctx)
	if err != nil && err != io.EOF {
		log.WithError(err).Error("listing bucket")
	}

	if config.Iface == "" {
		config.Iface = "127.0.0.1"
	}
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", config.Iface, config.Port)}
	res := &Proxy{
		bucket:   bucket,
		Server:   server,
		Backends: map[string]Backend{},
	}
	for k, v := range config.Backends {
		switch v.Type {
		case "mvn":
			var backend = NewMvnBackend(v)
			res.Backends[k] = backend
			break
		default:
			log.Panic("Unknown backend type:", v.Type)
		}
	}
	server.Handler = http.HandlerFunc(res.handler)
	return res, nil
}
