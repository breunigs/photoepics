package browser

import (
	md5mod "crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/peterbourgon/diskv"
)

const maxCacheSize = 264 * 1024 * 1024 // 264 MiB
const basePath = ".browserCache"

var expireTime = 30 * 24 * time.Hour // 1 month

var diskCache = diskv.New(diskv.Options{
	BasePath: ".browserCache",
	// Transform:    blockTransform,
	AdvancedTransform: advancedTransform,
	InverseTransform:  inverseTransform,
	CacheSizeMax:      maxCacheSize,
})

func WriteToCache(uri string, val []byte) error {
	return diskCache.Write(url2key(uri), val)
}

func ReadFromCache(uri string) string {
	return diskCache.ReadString(url2key(uri))
}

func Expire() {
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return
	}

	expireBefore := time.Now().Add(-expireTime)

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("cannot walk cache dir %q: %v\n", basePath, err)
		}
		if !info.IsDir() && info.ModTime().Before(expireBefore) {
			log.Printf("Epxiring %s (created: %s)", path, info.ModTime())
			return os.Remove(path)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("cannot walk cache dir %q: %v\n", basePath, err)
	}
}

func advancedTransform(key string) *diskv.PathKey {
	slice := strings.Split(key, "__")
	last := len(slice) - 1
	return &diskv.PathKey{
		Path:     slice[:last],
		FileName: slice[last],
	}
}

func inverseTransform(pathKey *diskv.PathKey) (key string) {
	return strings.Join(pathKey.Path, "__") + "__" + pathKey.FileName
}

// func blockTransform(s string) []string {
// 	slice := strings.Split(s, "__")
// 	return slice[:len(slice)-1]
// }

func url2key(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	path := re.ReplaceAllString(u.Path, "")

	return fmt.Sprintf("%s__%s__%s", u.Host, path, md5(uri))
}

func md5(text string) string {
	hash := md5mod.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
