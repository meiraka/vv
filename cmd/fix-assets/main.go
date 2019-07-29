package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/dave/jennifer/jen"
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
	gz := map[string][]byte{}
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
		gz[p] = g
	}

	f := jen.NewFile("main")
	c := make([]jen.Code, 0, len(gz))
	for p, v := range gz {
		c = append(c, jen.Id(makeName(p)).Op("=").Index().Byte().Parens(jen.Lit(string(v))))
	}
	f.Var().Defs(c...)

	fmt.Printf("%#v", f)
}
