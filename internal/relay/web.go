package relay

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed web/receive.html
var receiveHTML []byte

func (r *Relay) HandleWeb(w http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, "/d/") {
		http.NotFound(w, req)
		return
	}
	if !r.checkAccess(w, req) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(receiveHTML)
}
