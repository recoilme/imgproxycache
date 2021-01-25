package main

import (
	"context"
	"crypto/md5"
	"html"

	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/recoilme/graceful"
)

var (
	errURLLoad    = "Error: load url"
	errCacheShort = "Error: hash to short"
)

func main() {
	// config load
	address := ":8081"

	// signal check
	quit := make(chan os.Signal, 1)
	graceful.Unignore(quit, fallback, graceful.Terminate...)
	// metrics

	// service
	http.HandleFunc("/", mainPage)

	log.Fatal(http.ListenAndServe(address, nil))
}

func fallback() error {

	fmt.Println("Bye", time.Now())
	return nil
}

// imgLoad load image data
func imgLoad(imgURL string) ([]byte, error) {
	log.Println("imgLoad", imgURL)
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*10)
	defer cncl()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		//log.Println(err, imgUrl)
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err, imgURL)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		fmt.Println(err, imgURL)
		return nil, errors.New(errURLLoad)
	}

	bin, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err, imgURL)
		return nil, err
	}

	buf := make([]byte, 512)
	copy(buf, bin)
	cntType := strings.ToLower(http.DetectContentType(buf))
	//log.Println(cntType)
	if !strings.HasPrefix(cntType, "image") {
		return nil, errors.New(errURLLoad)
	}
	return bin, nil
}

// md5 hash
// no error checks here (md5 will return something for any data)
func hash(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// cachePut - get hash from imgURL and save imgData in file like:
//c/29/b7f54b2df7773722d382f4809d650
// return hash, error
func cachePut(imgURL string, imgData []byte) (string, error) {
	log.Println("cachePut", imgURL)
	hash := hash(imgURL)
	err := cacheMkDirs(hash)
	if err != nil {
		return "", err
	}
	filePath, err := cacheFilePath(hash)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(filePath, imgData, 0644)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// cacheMkDirs create dirs s/ss from any string
func cacheMkDirs(s string) error {
	if len(s) <= 3 {
		return errors.New(errCacheShort)
	}
	dir := fmt.Sprintf("%s/%s", string(s[0]), string(s[1:3]))
	return os.MkdirAll(dir, 0755)
}

// cacheFilePath convert string like c29b7f54b2df7773722d382f4809d650
// to string like c/29/b7f54b2df7773722d382f4809d650
func cacheFilePath(hash string) (string, error) {
	if len(hash) <= 4 {
		return "", errors.New(errCacheShort)
	}
	return fmt.Sprintf("%s/%s/%s", string(hash[0]), string(hash[1:3]), string(hash[4:])), nil
}

/*
//PKG_CONFIG_PATH="$(brew --prefix libffi)/lib/pkgconfig"   CGO_LDFLAGS_ALLOW="-s|-w"   CGO_CFLAGS_ALLOW="-Xpreprocessor" go build
func resize(s string) error {
	w := 776
	h := 416
	options := bimg.Options{
		Width:     w,
		Height:    h,
		Crop:      true,
		Quality:   95,
		Gravity:   bimg.GravitySmart,
		Interlace: true,
	}

	filePath, err := cacheFilePath(s)
	if err != nil {
		return err
	}

	buffer, err := bimg.Read(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	newImage, err := bimg.NewImage(buffer).Process(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	bimg.Write("new.jpg", newImage)

	options.Gravity = bimg.GravityCentre
	newImage, err = bimg.NewImage(buffer).Process(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	bimg.Write("new2.jpg", newImage)

	return nil
}
*/

// cacheDelWithDirs delete cache file
// if withDirs==true - remove dirs if not empty
func cacheDelWithDirs(s string, withDirs bool) error {
	filePath, err := cacheFilePath(hash(s))
	if err != nil {
		return err
	}
	err = os.Remove(filePath)
	if err != nil {
		return err
	}
	if !withDirs {
		return nil
	}
	err = os.Remove(string(filePath[:4])) // remove subdir: s/ss
	if err != nil {
		return err
	}
	err = os.Remove(filePath[:2]) // remove dir /s
	return err
}

// cacheGet - convert string to hash, to hashPath and read
// return error if any
func cacheGet(imgURL string) ([]byte, error) {
	log.Println("cacheGet", imgURL)
	hash := hash(imgURL)
	filePath, err := cacheFilePath(hash)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(filePath)
}

//mainPage handler
func mainPage(w http.ResponseWriter, r *http.Request) {
	path := html.EscapeString(r.URL.Path)
	log.Println("mainPage", path)
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
			w.WriteHeader(404)
			return
		}
		buf := make([]byte, 512)
		copy(buf, bin)
		cntType := strings.ToLower(http.DetectContentType(buf))
		if !strings.HasPrefix(cntType, "image") {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", cntType)
		w.Write(bin)
	default:
		w.WriteHeader(503)
	}
}

// urlGet - check cache, load and put in cache if not
func urlGet(imgURL string) ([]byte, error) {
	log.Println("urlGet", imgURL)
	// check cache
	bin, err := cacheGet(imgURL)
	if err == nil {
		log.Println("get from cache", imgURL)
		return bin, err
	}
	// load
	bin, err = imgLoad(imgURL)
	if err != nil {
		return nil, err
	}
	//put in cache
	_, err = cachePut(imgURL, bin)
	if err != nil {
		return nil, err
	}
	return bin, nil
}
