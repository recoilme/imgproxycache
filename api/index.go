package handler

import (
	"net/http"

	"github.com/recoilme/imgproxycache/imgcache"
)

// Handler for vercel.com
// test with: https://imgproxycache.vercel.app/api/index.go?url=https://i.trbna.com/preset/wysiwyg/2/61/0e0984cf811eb927ad03e1d88bea4.jpeg
func Handler(w http.ResponseWriter, r *http.Request) {
	imgcache.MainPageNoCache(w, r)
}
