package app

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zorcal/its-a-me-zorcal/pkg/httprouter"
)

func staticHandler(static fs.FS, appVersion string, disableCache bool) httprouter.Handler {
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(static)))

	return httprouter.HandlerFromStd(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if disableCache {
			staticHandler.ServeHTTP(w, r)
			return
		}

		filePath := strings.TrimPrefix(r.URL.Path, "/static/")
		maxAge := calcStaticFileMaxAge(filePath)

		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(maxAge.Seconds())))
		w.Header().Set("Expires", time.Now().Add(maxAge).Format(http.TimeFormat))

		if fileInfo, err := fs.Stat(static, filePath); err == nil {
			etag := fmt.Sprintf(`"%x"`, md5.Sum([]byte(appVersion+filePath+fileInfo.ModTime().String())))
			w.Header().Set("ETag", etag)

			if match := r.Header.Get("If-None-Match"); match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		staticHandler.ServeHTTP(w, r)
	}))
}

func calcStaticFileMaxAge(filePath string) time.Duration {
	switch {
	case strings.HasSuffix(filePath, ".css") || strings.HasSuffix(filePath, ".js"):
		return 7 * 24 * time.Hour // 7 days for CSS/JS
	default:
		return 24 * time.Hour // 1 day for others
	}
}
