package assets

import (
	"crypto/md5"
	"embed"
	"encoding/hex"
	"fmt"
	"mime"
	"path"
	"strconv"

	"github.com/meiraka/vv/internal/gzip"
)

func init() {
	if err := initEmbed(); err != nil {
		panic(fmt.Errorf("assets: %w", err))
	}
}

//go:embed app-black.* app.* manifest.json  nocover.svg  w.png
var embedFS embed.FS
var (
	embedPath     []string
	embedBody     [][]byte
	embedLength   []string
	embedHash     []string
	embedGZBody   [][]byte
	embedGZLength []string
	embedGZHash   []string
	embedMine     []string
)

func initEmbed() error {
	dir, err := embedFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("embed: readdir: %w", err)
	}
	length := len(dir)
	embedPath = make([]string, length)
	embedBody = make([][]byte, length)
	embedLength = make([]string, length)
	embedHash = make([]string, length)
	embedGZBody = make([][]byte, length)
	embedGZLength = make([]string, length)
	embedGZHash = make([]string, length)
	embedMine = make([]string, length)

	for i, f := range dir {
		if f.IsDir() {
			continue
		}
		embedPath[i] = path.Join("/", "assets", f.Name())
		m := mime.TypeByExtension(path.Ext(f.Name()))
		b, err := embedFS.ReadFile(f.Name())
		if err != nil {
			return fmt.Errorf("embed: readfile: %w", err)
		}
		embedBody[i] = b
		embedLength[i] = strconv.Itoa(len(b))
		hasher := md5.New()
		hasher.Write(b)
		embedHash[i] = hex.EncodeToString(hasher.Sum(nil))
		if m != "image/png" && m != "image/jpg" {
			gz, err := gzip.Encode(b)
			if err != nil {
				return fmt.Errorf("%s: gzip: %w", f.Name(), err)
			}
			embedGZBody[i] = gz
			embedGZLength[i] = strconv.Itoa(len(gz))
			hasher := md5.New()
			hasher.Write(gz)
			embedGZHash[i] = hex.EncodeToString(hasher.Sum(nil))
		}
		embedMine[i] = m
	}
	return nil
}

func embedIndex(path string) (index int, ok bool) {
	for i := range embedPath {
		if path == embedPath[i] {
			return i, true
		}
	}
	return -1, false
}

// Hash returns embeded assets file MD5 hash.
func Hash(path string) (string, bool) {
	i, ok := embedIndex(path)
	if !ok {
		return "", false
	}
	return embedHash[i], true
}
