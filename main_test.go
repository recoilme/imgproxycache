package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/muesli/smartcrop"
	"github.com/muesli/smartcrop/nfnt"
	"github.com/stretchr/testify/assert"
)

var (
	img1 = "https://yastatic.net/s3/home-static/_/x/Q/xk8YidkhGjIGOrFm_dL5781YA.svg"
	img2 = "https://4.bp.blogspot.com/-LexDP8Xmkf4/TseIm5fXRiI/AAAAAAAAB-s/ppI8XLU48e4/s1600/Emma+Watson+III.jpg"
	img3 = "https://3.bp.blogspot.com/-sNCiRqcSIQU/UzV0UryGpAI/AAAAAAAATJU/dLK-UJLsgCU/s3800/Emma+Watson+photo.filmcelebritiesactresses.blogspot-005.jpg"
	img4 = "https://bestofcomicbooks.com/wp-content/uploads/2018/09/emma-watson-smile.jpg"
	img5 = "https://upload.wikimedia.org/wikipedia/commons/thumb/d/d0/Northern_Spotted_Owl.USFWS.jpg/1200px-Northern_Spotted_Owl.USFWS.jpg"
	img6 = "https://nas-national-prod.s3.amazonaws.com/h_a1_5870_1_snowy-owl_robert_holden_immature.jpg"
)

func TestImgLoad(t *testing.T) {
	// svg not an image - error
	bin, err := imgLoad(img1)
	assert.Error(t, err)
	assert.Equal(t, 0, len(bin))

	//jpg is image - no error
	bin, err = imgLoad(img2)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(bin))
	h, err := cachePut(img2, bin)
	fmt.Println(h)
}

func TestCachePut(t *testing.T) {
	someString := "any"
	buf := make([]byte, 1)
	buf[0] = 42
	cacheName, err := cachePut(someString, buf)
	assert.NoError(t, err)
	assert.NotEqual(t, "", cacheName)

	err = cacheDelWithDirs(someString, true)
	assert.NoError(t, err)
}

func TestCacheFilePath(t *testing.T) {
	res, err := cacheFilePath("")
	assert.Error(t, err)
	assert.Equal(t, "", res)
}

func TestCacheGet(t *testing.T) {
	someString := "any"
	buf := make([]byte, 1)
	buf[0] = 42
	cacheName, err := cachePut(someString, buf)
	assert.NoError(t, err)
	assert.NotEqual(t, "", cacheName)

	bin, err := cacheGet(someString)
	assert.NoError(t, err)
	assert.Equal(t, "*", string(bin)) //string(42)=="*"
	err = cacheDelWithDirs(someString, true)
	assert.NoError(t, err)
}

func TestUrlGet(t *testing.T) {
	imgURL := img2
	bin, err := urlGet(imgURL)
	assert.NoError(t, err)
	assert.NotNil(t, bin)
	cacheDelWithDirs(imgURL, true)
	assert.NoError(t, err)

	//put to cache
	bin, err = urlGet(imgURL)
	assert.NoError(t, err)
	assert.NotNil(t, bin)
	//get from cache
	bin, err = urlGet(imgURL)
	assert.NoError(t, err)
	assert.NotNil(t, bin)
	cacheDelWithDirs(imgURL, true)
	assert.NoError(t, err)
}

func TestMainHandler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(mainPage))
	defer ts.Close()
	href := img2

	imgURL := fmt.Sprintf("%s/?url="+href, ts.URL)
	res, err := http.Get(imgURL)
	assert.NoError(t, err)
	assert.Equal(t, res.StatusCode, 200)
	defer res.Body.Close()
	bin, err := ioutil.ReadAll(res.Body)

	assert.NoError(t, err)
	assert.NotNil(t, bin)

	href = strings.ReplaceAll(href, "+", " ")
	cacheDelWithDirs(href, true)
	assert.NoError(t, err)
}

//https://29a.ch/sandbox/2014/smartcrop/examples/testbed.html
func TestSc(t *testing.T) {
	type SubImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	w := 776
	h := 620 //416
	imgURL := "https://github.com/muesli/smartcrop-samples/blob/master/muesli/guitarist.jpg?raw=true"
	//"https://i.trbna.com/preset/wysiwyg/2/61/0e0984cf811eb927ad03e1d88bea4.jpeg"
	bin, err := urlGet(imgURL)
	assert.NoError(t, err)
	assert.NotNil(t, bin)
	analyzer := smartcrop.NewAnalyzer(nfnt.NewDefaultResizer())
	img, _, err := image.Decode(bytes.NewReader(bin))
	assert.NoError(t, err)
	rect, err := analyzer.FindBestCrop(img, w, h)
	assert.NoError(t, err)

	sub, ok := img.(SubImager)
	if ok {
		cropImage := sub.SubImage(rect)
		// cropImage := sub.SubImage(image.Rect(topCrop.X, topCrop.Y, topCrop.Width+topCrop.X, topCrop.Height+topCrop.Y))
		smartcrop.WriteImage("jpeg", cropImage, "./smartcrop.jpg")
	} else {
		t.Error(errors.New("No SubImage support"))
	}
}

/*
func TestResize(t *testing.T) {
	bin, err := imgLoad("https://i.trbna.com/preset/wysiwyg/2/61/0e0984cf811eb927ad03e1d88bea4.jpeg")
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(bin))
	h, err := cachePut(img3, bin)
	resize(h)
}

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
