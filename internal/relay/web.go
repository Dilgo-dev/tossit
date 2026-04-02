package relay

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web/dist
var webDist embed.FS

func (r *Relay) WebHandler() http.Handler {
	if !r.cfg.UIEnabled {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/d/") {
				http.NotFound(w, req)
				return
			}
			http.NotFound(w, req)
		})
	}

	dist, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/d/") {
			code := strings.TrimPrefix(req.URL.Path, "/d/")
			target := "/#/receive/" + code
			if q := req.URL.RawQuery; q != "" {
				target += "?" + q
			}
			http.Redirect(w, req, target, http.StatusFound)
			return
		}

		if !r.checkAccess(w, req) {
			return
		}

		if !r.checkUIPassword(w, req) {
			return
		}

		path := req.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		if _, err := fs.Stat(dist, strings.TrimPrefix(path, "/")); err == nil {
			fileServer.ServeHTTP(w, req)
			return
		}

		req.URL.Path = "/"
		fileServer.ServeHTTP(w, req)
	})
}

func (r *Relay) HandleConfig(w http.ResponseWriter, req *http.Request) {
	if !r.checkAccess(w, req) {
		return
	}
	if !r.checkAdminPassword(w, req) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"port":%q,"storage":%q,"expire":%q,"max_size":%q,"rate_limit":%d}`,
		r.cfg.Port,
		r.cfg.StorageDir,
		r.cfg.Expire.String(),
		FormatSize(r.cfg.MaxSize),
		r.cfg.RateLimit,
	)
}
