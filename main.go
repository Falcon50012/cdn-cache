package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

var cacheDir = "tmp/cdn_cache/"

type content struct {
	target   string
	fileName string
	maxSize  int
	// cacheMap map[string]*cache
	// mu       sync.Mutex
}

// type cache struct {
// 	isCached   bool
// 	lastUpdate time.Time
// }

func newContent() *content {
	return &content{
		maxSize: 10 * 1024 * 1024,
		// cacheMap: map[string]*cache{},
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

		c.fileName = hashFileName(c)
		_, err := os.Stat(cacheDir + c.fileName)
		if err != nil && errors.Unwrap(err).Error() != "The system cannot find the file specified." {
			fmt.Println(err)
			return
		}
		if err != nil && errors.Unwrap(err).Error() == "The system cannot find the file specified." {
			_, err := downloadContent(c)
			if err != nil {
				http.Error(w, "Failed to download file", http.StatusInternalServerError)
				return
			}
		}

		http.ServeFile(w, r, cacheDir+c.fileName)
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
	// fmt.Printf("File size: %s byte\n", contentLength)

	intContentLength, _ := strconv.Atoi(contentLength)
	if intContentLength > c.maxSize {
		log.Println("File too large. Cannot download it")
		return nil, err
	}

	outFile, err := os.Create(cacheDir + c.fileName)
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

	fileStat, _ := outFile.Stat()
	fmt.Printf("FileName: %v\nSize: %v bytes\nMode: %v\nModTime: %v\nIsDir: %v\nSys: %v\n",
		fileStat.Name(), fileStat.Size(), fileStat.Mode(), fileStat.ModTime(), fileStat.IsDir(), fileStat.Sys())

	println("Download successful")

	return outFile, nil
}
