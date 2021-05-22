package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
)

func makeName(p string) string {
	n := make([]rune, 0, len(p))
	title := true
	up := false
	for _, v := range p {
		if 97 <= v && v <= 122 {
			if title || up {
				v -= 32
			}
			n = append(n, v)
			title = false
		} else if 65 <= v && v <= 90 {
			if !title && !up {
				v += 32
			}
			n = append(n, v)
			title = false
		} else if 48 <= v && v <= 57 {
			if len(n) != 0 {
				n = append(n, v)
			}
		} else if v == 46 {
			up = true
			title = false
		} else {
			title = true
			up = false
		}
	}
	return string(n)
}

func main() {
	files, err := ioutil.ReadDir("assets")
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	f, err := os.Create(filepath.Join("internal", "vv", "assets", "binary.go"))
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	fmt.Fprintln(f, "package assets")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "var (")
	for _, file := range files {
		p := path.Join("assets", file.Name())
		b, err := ioutil.ReadFile(p)
		if err != nil {
			log.Fatalf("failed to read %s: %v", p, err)
		}
		hasher := md5.New()
		hasher.Write(b)
		h := hex.EncodeToString(hasher.Sum(nil))
		name := makeName(file.Name())
		fmt.Fprintf(f, "\t// %s is %s\n", name, p)
		fmt.Fprintf(f, "\t%s = []byte(%q)\n", name, b)
		fmt.Fprintf(f, "\t// %sHash is md5 for %s\n", name, p)
		fmt.Fprintf(f, "\t%sHash = []byte(%q)\n", name, h)
	}
	fmt.Fprintln(f, ")")
}
