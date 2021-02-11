package imgcache

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var (
	//errors
	errURLLoad    = "Error: load url"
	errCacheShort = "Error: hash to short"
	// metrics
	reqSuccess uint64
	reqError   uint64
)

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

// md5 hash
// no error checks here (md5 will return something for any data)
func hash(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// cacheFilePath convert string like c29b7f54b2df7773722d382f4809d650
// to string like c/29/b7f54b2df7773722d382f4809d650
func cacheFilePath(hash string) (string, error) {
	if len(hash) <= 4 {
		return "", errors.New(errCacheShort)
	}
	return fmt.Sprintf("%s/%s/%s", string(hash[0]), string(hash[1:3]), string(hash[4:])), nil
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

func writeImage(imgtype string, img image.Image, name string) error {
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		panic(err)
	}

	switch imgtype {
	case "png":
		return writeImageToPng(img, name)
	case "jpeg":
		return writeImageToJpeg(img, name)
	}

	return errors.New("Unknown image type")
}

func writeImageToJpeg(img image.Image, name string) error {
	fso, err := os.Create(name)
	if err != nil {
		return err
	}
	defer fso.Close()

	return jpeg.Encode(fso, img, &jpeg.Options{Quality: 100})
}

func writeImageToPng(img image.Image, name string) error {
	fso, err := os.Create(name)
	if err != nil {
		return err
	}
	defer fso.Close()

	return png.Encode(fso, img)
}
