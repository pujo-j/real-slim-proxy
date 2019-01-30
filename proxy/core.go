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
	l, err := li.Next(ctx)
	if err != nil && err != io.EOF {
		log.WithError(err).Error("listing bucket")
	}

	log.WithField("listObject", l).Debug("First file found")
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", config.Port)}
	res := &Proxy{
		bucket:   bucket,
		Server:   server,
		Backends: map[string]Backend{},
	}
	server.Handler = http.HandlerFunc(res.handler)
	return res, nil
}
