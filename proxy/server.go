package proxy

import (
	"fmt"
	"net/http"
)

func (proxy Proxy) handler(w http.ResponseWriter, r *http.Request) {
	//for _, backend := range proxy.Backends {

	//}
	w.Header().Set("Content-Type", "text/text")
	_, _ = w.Write(
		[]byte(fmt.Sprintf("%#v\n", r)))
	_, _ = w.Write(
		[]byte(fmt.Sprintf("%#v\n", proxy.bucket)))
}
