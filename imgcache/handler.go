package imgcache

import (
	"fmt"
	"html"
	"net/http"
)

func ImgHandler(w http.ResponseWriter, r *http.Request) {
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

		fmt.Fprintf(w, "<h1>Hello world, url = "+url+"</h1>")

		return
	default:
		w.WriteHeader(503)
		return
	}
}
