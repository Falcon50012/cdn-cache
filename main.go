package main

import (
	"crypto/sha256"
	"encoding/hex"
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
	cacheMap map[string]*cache
	mu       sync.Mutex // * || !* ?
}

type cache struct {
	filePath   string
	lastUpdate time.Time
}

func newContent() *content {
	return &content{
		maxSize:  10 * 1024 * 1024,
		cacheMap: map[string]*cache{},
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
			return
		}
	})
}

func hashFileName(c *content) string {
	hash := sha256.New()
	hash.Write([]byte(c.target))
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	// fmt.Println("hashString:", hashString)
	return hashString
}

func downloadContent(c *content) (*os.File, error) {
	// var outFile *os.File

	response, err := http.Get(c.target)
	if err != nil {
		log.Println("GET request error:", err)
		return nil, err
	}
	defer response.Body.Close()

	fileName := hashFileName(c)

	// if c.isCached {
	// 	log.Println("File is already in cache")
	// 	return outFile, nil
	// }

	contentLength := response.Header.Get("Content-Length")
	if contentLength == "" {
		fmt.Println("Failed to get file size.")
		return nil, err
	}
	// fmt.Printf("File size: %s byte\n", contentLength)

	intContentLength, _ := strconv.Atoi(contentLength)
	if intContentLength > c.maxSize {
		log.Println("File too large. Cannot download it")
		return nil, err
	}

	contentType := response.Header.Get("Content-Type")
	// fmt.Println("Content-Type:", contentType)

	fileType := strings.SplitAfter(contentType, "/")
	extension := "." + fileType[1]
	// fmt.Println("Extension:", extension)

	// fileName := time.Now().Format("2006-01-02-15-04-05")
	// fmt.Println(fileName)

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

	// fmt.Println("outFile.Name()", outFile.Name())
	fileStat, _ := outFile.Stat()
	fmt.Printf("FileName: %v\nSize: %v bytes\nMode: %v\nModTime: %v\nIsDir: %v\nSys: %v\n",
		fileStat.Name(), fileStat.Size(), fileStat.Mode(), fileStat.ModTime(), fileStat.IsDir(), fileStat.Sys())

	println("Download successful")

	// c.isCached = true
	// log.Printf("File %s is cached\n", fileStat.Name())

	return outFile, nil
}

func storeInMap(c *content) {

}
