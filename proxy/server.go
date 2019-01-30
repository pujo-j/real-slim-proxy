package proxy

import (
	"net/http"
	"strings"
)

func (proxy Proxy) handler(w http.ResponseWriter, r *http.Request) {
	splitPath := strings.Split(r.URL.Path, "/")
	if len(splitPath) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	prefix := splitPath[1]
	path := strings.Join(splitPath[2:], "/")
	for _, backend := range proxy.Backends {
		if strings.EqualFold(prefix, backend.GetConfig().Prefix) {
			backend.GetResource(path, w, r, proxy.bucket)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}
