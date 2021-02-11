package main

import (
	"flag"

	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/recoilme/graceful"
	"github.com/recoilme/imgproxycache/imgcache"
)

var (
	//params
	address = flag.String("address", ":8081", "address to listen on (default: :8081)")
)

func main() {
	// config load
	flag.Parse()

	// signal check
	quit := make(chan os.Signal, 1)
	graceful.Unignore(quit, fallback, graceful.Terminate...)

	// service
	http.HandleFunc("/", imgcache.MainPage)
	fmt.Println(http.ListenAndServe(*address, nil))
}

func fallback() error {

	fmt.Println("Bye", time.Now())
	return nil
}
