package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"sort"
)

func makeGZip(data []byte) ([]byte, error) {
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return gz.Bytes(), nil
}

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
	fmt.Println("package main")
	fmt.Println()
	fmt.Println("var (")
	for _, file := range files {
		p := path.Join("assets", file.Name())
		b, err := ioutil.ReadFile(p)
		if err != nil {
			log.Fatalf("failed to read %s: %v", p, err)
		}
		g, err := makeGZip(b)
		if err != nil {
			log.Fatalf("failed to make gzip %s: %v", p, err)
		}
		fmt.Printf("\t// %s is gzip encoded %s\n", makeName(p), p)
		fmt.Printf("\t%s = []byte(%q)\n", makeName(p), g)
	}
	fmt.Println(")")
}
