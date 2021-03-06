package imgcache

import (
	"html"
	"net/http"
	"strings"
	"sync/atomic"
)

//func Handler(w http.ResponseWriter, r *http.Request) {
//	MainPage(w, r)
//}

//MainPage handler
func MainPage(w http.ResponseWriter, r *http.Request) {
	path := html.EscapeString(r.URL.Path)
	log("mainPage", path)
	switch r.Method {
	case "GET":
		vals := r.URL.Query()
		url := vals.Get("url")
		if path == "/" && url == "" {
			w.WriteHeader(200)
			return
		}

		bin, err := urlGet(url)
		if err != nil {
			log("urlGet err", err)
			atomic.AddUint64(&reqError, 1)
			w.WriteHeader(404)
			return
		}
		buf := make([]byte, 512)
		copy(buf, bin)
		cntType := strings.ToLower(http.DetectContentType(buf))
		if !strings.HasPrefix(cntType, "image") {
			log("not image err", url)
			w.WriteHeader(404)
			atomic.AddUint64(&reqError, 1)
			return
		}
		w.Header().Set("Content-Type", cntType)
		atomic.AddUint64(&reqSuccess, 1)
		w.Write(bin)
		return
	default:
		log("wrong params")
		w.WriteHeader(503)
		return
	}
}

// MainPageNoCache - load image on the fly
func MainPageNoCache(w http.ResponseWriter, r *http.Request) {
	path := html.EscapeString(r.URL.Path)
	log("mainPage", path)
	switch r.Method {
	case "GET":
		vals := r.URL.Query()
		url := vals.Get("url")
		if path == "/" && url == "" {
			w.WriteHeader(200)
			return
		}

		bin, err := imgLoad(url)
		if err != nil {
			log("imgLoad err", err)
			atomic.AddUint64(&reqError, 1)
			w.WriteHeader(404)
			return
		}
		buf := make([]byte, 512)
		copy(buf, bin)
		cntType := strings.ToLower(http.DetectContentType(buf))
		if !strings.HasPrefix(cntType, "image") {
			log("not image err", url)
			w.WriteHeader(404)
			atomic.AddUint64(&reqError, 1)
			return
		}
		w.Header().Set("Content-Type", cntType)
		atomic.AddUint64(&reqSuccess, 1)
		w.Write(bin)
		return
	default:
		log("wrong params")
		w.WriteHeader(503)
		return
	}
}
