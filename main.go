package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"flag"
	"html"
	"reflect"
	"sync/atomic"

	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/recoilme/graceful"
)

var (
	//errors
	errURLLoad    = "Error: load url"
	errCacheShort = "Error: hash to short"
	// metrics
	reqSuccess uint64
	reqError   uint64
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
	http.HandleFunc("/", mainPage)
	fmt.Println(http.ListenAndServe(*address, nil))
}

func fallback() error {

	fmt.Println("Bye", time.Now())
	return nil
}

// imgLoad load image data
func imgLoad(imgURL string) ([]byte, error) {
	log("imgLoad", imgURL)
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*10)
	defer cncl()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		//log.Println(err, imgUrl)
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log(err, imgURL)
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
	log("cachePut", imgURL)
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
	log("cacheGet", imgURL)
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

// urlGet - check cache, load and put in cache if not
func urlGet(imgURL string) ([]byte, error) {
	log("urlGet", imgURL)
	// check cache
	bin, err := cacheGet(imgURL)
	if err == nil {
		log("get from cache", imgURL)
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

func log(a ...interface{}) (n int, err error) {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("%s ", time.Now().Format("15:04:05")))
	isError := false
	for _, s := range a {
		buf.WriteString(fmt.Sprint(s, " "))
		if strings.HasPrefix(reflect.TypeOf(a).String(), "*error") {
			isError = true
		}
	}
	if isError /*&& graylog != nil*/ {
		//graylog.Error(buf.String())
		//fmt.Println(os.Stderr, buf.String())
	}
	return fmt.Fprintln(os.Stdout, buf.String())
}
