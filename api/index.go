package handler

import (
	"net/http"

	"github.com/recoilme/imgproxycache/imgcache"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	//Img(w, r)
	imgcache.MainPage(w, r)
}
