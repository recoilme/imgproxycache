package imgcache

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync/atomic"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	path := html.EscapeString(r.URL.Path)
	//log("mainPage", path)
	switch r.Method {
	case "GET":
		vals := r.URL.Query()
		url := vals.Get("url")
		if path == "/" && url == "" {
			w.WriteHeader(200)
			return
		}

		fmt.Fprintf(w, "<h1>Hello, url = "+url+"</h1>")

		return
	default:
		w.WriteHeader(503)
		return
	}
}

//mainPage handler
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
			atomic.AddUint64(&reqError, 1)
			w.WriteHeader(404)
			return
		}
		buf := make([]byte, 512)
		copy(buf, bin)
		cntType := strings.ToLower(http.DetectContentType(buf))
		if !strings.HasPrefix(cntType, "image") {
			w.WriteHeader(404)
			atomic.AddUint64(&reqError, 1)
			return
		}
		w.Header().Set("Content-Type", cntType)
		atomic.AddUint64(&reqSuccess, 1)
		w.Write(bin)
		return
	default:
		w.WriteHeader(503)
		return
	}
}
