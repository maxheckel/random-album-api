package main

import (
	"encoding/json"
	"fmt"
	"github.com/EdlinOrg/prominentcolor"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
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

func HandleGetColors(w http.ResponseWriter, r *http.Request){
	fullUrlFile = r.FormValue("image_url")

	// Build fileName from fullPath
	buildFileName()

	// Create blank file
	file := createFile()

	// Put content on file
	putFile(file, httpClient())
	existingImageFile, err := os.Open(file.Name())

	defer existingImageFile.Close()
	defer os.Remove(file.Name())

	// Calling the generic image.Decode() will tell give us the data
	// and type of image it is as a string. We expect "png"
	imageData, _, err := image.Decode(existingImageFile)
	checkError(err)
	res, err := prominentcolor.Kmeans(imageData)
	checkError(err)
	resp := &ColorResponse{}
	resp.Colors = append(resp.Colors, "#"+res[0].AsString())
	resp.Colors = append(resp.Colors, "#"+res[1].AsString())
	resp.Colors = append(resp.Colors, "#"+res[2].AsString())
	colorsJson, err := json.Marshal(resp)
	checkError(err)
	w.Write(colorsJson)

}

type ColorResponse struct{
	Colors []string `json:"colors"`
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
