package handler

import (
	"net/http"

	"github.com/recoilme/imgproxycache/imgcache"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	imgcache.MainPage(w, r)
}
