package main

import (
	"encoding/json"
	"fmt"
	"github.com/EdlinOrg/prominentcolor"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"image"
	"image/color"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

var (
	fileName    string
	fullUrlFile string
)

func main() {
	r := mux.NewRouter()

	// IMPORTANT: you must specify an OPTIONS method matcher for the middleware to set CORS headers
	r.HandleFunc("/colors", HandleGetColors).Methods(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodOptions)
	r.Use(mux.CORSMethodMiddleware(r))

	http.ListenAndServe(":8081", handlers.CORS()(r))

}

func HandleGetColors(w http.ResponseWriter, r *http.Request) {
	fullUrlFile = r.FormValue("image_url")

	// Build fileName from fullPath
	buildFileName()

	// Create blank file
	file := createFile()

	// Put content on file
	putFile(file, httpClient())
	existingImageFile, err := os.Open(file.Name())
	checkError(err)
	defer existingImageFile.Close()
	defer os.Remove(file.Name())


	// Calling the generic image.Decode() will tell give us the data
	// and type of image it is as a string. We expect "png"
	imageData, _, err := image.Decode(existingImageFile)
	_, err = prominentcolor.Kmeans(imageData)
	//fmt.Println(imageData.Bounds())
	//smallerImage := resize.Resize(uint(imageData.Bounds().Max.X/2), uint(imageData.Bounds().Max.Y/2), imageData, resize.Lanczos3)
	quadrents := GetImageQuadrents(imageData)
	sort.Slice(quadrents, func(i, j int) bool {
		hsl1 := ToHSL(quadrents[i])
		hsl2 := ToHSL(quadrents[j])
		return hsl1.H < hsl2.H
	})
	resp := &ColorResponse{}
	resp.Colors = append(resp.Colors, fmt.Sprintf("#%.2x%.2x%.2x", quadrents[3].R, quadrents[3].G, quadrents[3].B))
	resp.Colors = append(resp.Colors, fmt.Sprintf("#%.2x%.2x%.2x", quadrents[11].R, quadrents[11].G, quadrents[11].B))


	//resp.Colors = append(resp.Colors, "#"+res[2].AsString())
	colorsJson, err := json.Marshal(resp)
	checkError(err)
	w.Write(colorsJson)

}

func GetImageQuadrents(i image.Image) []color.NRGBA {
	bounds := i.Bounds()
	var colorsList []color.NRGBA
	for q := float64(0); q < 4; q++ {
		minY := math.Floor(float64(bounds.Max.Y) * (q * 0.25))
		maxY := math.Floor(float64(bounds.Max.Y) * ((q + 1) * 0.25))

		for j := float64(0); j < 4; j++ {
			minX := math.Floor(float64(bounds.Max.X) * (j * 0.25))
			maxX := math.Floor(float64(bounds.Max.X) * ((j + 1) * 0.25))

			r := uint32(0)
			g := uint32(0)
			b := uint32(0)
			for y := minY; y < maxY; y++ {
				for x := minX; x < maxX; x++ {
					pr, pg, pb, _ := i.At(int(x), int(y)).RGBA()

					r += pr
					g += pg
					b += pb
				}
			}
			d := uint32(bounds.Dy() / 4 * bounds.Dx() / 4)
			r /= d
			g /= d
			b /= d

			colorsList = append(colorsList, color.NRGBA{uint8(r / 0x101), uint8(g / 0x101), uint8(b / 0x101), 255})
		}
	}
	return colorsList
}

type ColorResponse struct {
	Colors []string `json:"colors"`
}

type HSL struct {
	H, S, L float64
}

func (c HSL) ToRGB() color.RGBA {
	h := c.H
	s := c.S
	l := c.L

	if s == 0 {
		// it's gray
		return color.RGBA{uint8(l), uint8(l), uint8(l), 255}
	}

	var v1, v2 float64
	if l < 0.5 {
		v2 = l * (1 + s)
	} else {
		v2 = (l + s) - (s * l)
	}

	v1 = 2*l - v2

	r := hueToRGB(v1, v2, h+(1.0/3.0))
	g := hueToRGB(v1, v2, h)
	b := hueToRGB(v1, v2, h-(1.0/3.0))

	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}


func hueToRGB(v1, v2, h float64) float64 {
	if h < 0 {
		h += 1
	}
	if h > 1 {
		h -= 1
	}
	switch {
	case 6*h < 1:
		return (v1 + (v2-v1)*6*h)
	case 2*h < 1:
		return v2
	case 3*h < 2:
		return v1 + (v2-v1)*((2.0/3.0)-h)*6
	}
	return v1
}

func ToHSL(c color.NRGBA) HSL {
	var h, s, l float64

	r := float64(c.R)
	g := float64(c.G)
	b := float64(c.B)

	max := math.Max(math.Max(r, g), b)
	min := math.Min(math.Min(r, g), b)

	// Luminosity is the average of the max and min rgb color intensities.
	l = (max + min) / 2

	// saturation
	delta := max - min
	if delta == 0 {
		// it's gray
		return HSL{0, 0, l}
	}

	// it's not gray
	if l < 0.5 {
		s = delta / (max + min)
	} else {
		s = delta / (2 - max - min)
	}

	// hue
	r2 := (((max - r) / 6) + (delta / 2)) / delta
	g2 := (((max - g) / 6) + (delta / 2)) / delta
	b2 := (((max - b) / 6) + (delta / 2)) / delta
	switch {
	case r == max:
		h = b2 - g2
	case g == max:
		h = (1.0 / 3.0) + r2 - b2
	case b == max:
		h = (2.0 / 3.0) + g2 - r2
	}

	// fix wraparounds
	switch {
	case h < 0:
		h += 1
	case h > 1:
		h -= 1
	}

	return HSL{h, s, l}
}

func putFile(file *os.File, client *http.Client) {

	request, _ := http.NewRequest("GET", fullUrlFile, nil)

	request.Header.Set("User-Agent", "urvinyl.rocks")
	resp, err := client.Do(request)

	checkError(err)

	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)

	defer file.Close()

	checkError(err)

	fmt.Println("Just Downloaded a file %s with size %d", fileName, size)
}

func buildFileName() {
	fileUrl, err := url.Parse(fullUrlFile)
	checkError(err)

	path := fileUrl.Path
	segments := strings.Split(path, "/")

	fileName = segments[len(segments)-1]
}

func httpClient() *http.Client {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	return &client
}

func createFile() *os.File {
	file, err := os.Create(fileName)

	checkError(err)
	return file
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
