package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cacheDir = "tmp/cdn_cache/"

type content struct {
	target   string
	maxSize  int
	isCached bool
	cache    map[string]*cache
	mu       sync.Mutex // * || !* ?
}

type cache struct {
	filePath   string
	lastUpdate time.Time
}

func newContent() *content {
	return &content{
		maxSize: 10 * 1024 * 1024,
		cache:   map[string]*cache{},
	}
}

func main() {
	content := newContent()

	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		log.Fatal("Making directory error:", err)
	}

	http.Handle("/fetch", fetchHandler(content))

	fmt.Println("Server started at localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func fetchHandler(c *content) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.target = r.URL.Query().Get("url")
		if c.target == "" {
			http.Error(w, "Missing query parameter", http.StatusBadRequest)
			return
		}
		_, err := downloadContent(c)
		if err != nil {
			http.Error(w, "Failed to download file", http.StatusInternalServerError)
		}
	})
}

func downloadContent(c *content) (*os.File, error) {
	response, err := http.Get(c.target)
	if err != nil {
		log.Println("GET request error:", err)
		return nil, err
	}
	defer response.Body.Close()

	contentLength := response.Header.Get("Content-Length")
	if contentLength == "" {
		fmt.Println("Failed to get file size.")
		return nil, err
	}
	fmt.Printf("File size: %s byte\n", contentLength)

	intContentLength, _ := strconv.Atoi(contentLength)
	if intContentLength > c.maxSize {
		log.Println("File too large. Cannot download it")
		return nil, nil
	}

	contentType := response.Header.Get("Content-Type")
	fmt.Println("Content-Type:", contentType)

	fileType := strings.SplitAfter(contentType, "/")
	extension := "." + fileType[1]
	fmt.Println("Extension:", extension)

	fileName := time.Now().Format("2006-01-02-15-04-05")
	fmt.Println(fileName)

	outFile, err := os.Create(cacheDir + fileName + extension)
	if err != nil {
		log.Println("Blank file creation error", err)
		return nil, err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, response.Body)
	if err != nil {
		log.Println("Complete file creation error", err)
		return nil, err
	}

	println("Download successful")

	return outFile, nil
}
