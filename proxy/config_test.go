package proxy

import (
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		Port:  8086,
		Store: StoreConfig{Cloud: "gcs", Bucket: "test"},
	}
	out := &strings.Builder{}
	err := yaml.NewEncoder(out).Encode(config)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	t.Log(out.String())
}
