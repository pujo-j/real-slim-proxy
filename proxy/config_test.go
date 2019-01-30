package proxy

import (
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		Store: StoreConfig{Bucket: "test"},
		Backends: map[string]BackendConfig{
			"central": {
				Prefix:        "central",
				BaseUrl:       "http://central.maven.org/maven2/",
				CacheDuration: "2h",
			},
		},
	}
	config.applyDefaults()
	out := &strings.Builder{}
	err := yaml.NewEncoder(out).Encode(config)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	t.Log(out.String())
}
