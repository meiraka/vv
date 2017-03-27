// Code generated by go-bindata.
// sources:
// assets/app.css
// assets/app.html
// assets/app.js
// assets/next.svg
// assets/pause.svg
// assets/play.svg
// assets/prev.svg
// assets/random.svg
// assets/repeat.svg
// DO NOT EDIT!

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _assetsAppCss = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xbc\x57\xd9\x6e\xeb\x36\x10\x7d\xd7\x57\x10\x30\x02\xdc\x0b\x44\x86\x36\xbb\xb1\xfa\xde\xdf\x28\x28\x91\xb6\xd8\xd0\xa4\x40\x8d\x12\xe7\x16\xfe\xf7\x42\xd6\x62\xae\xb6\xd2\x02\x8d\x61\x20\x96\x38\x73\xce\x99\x85\x1c\x56\x92\x7c\xa1\xbf\x23\x84\x10\x8a\x3f\x69\xf5\xce\x20\x06\x7a\x81\xb8\x63\xbf\x68\x8c\xc9\x5f\x7d\x07\x25\x4a\x93\xe4\xe5\xf7\xdb\x9a\x33\x56\x27\x26\x4a\x94\xb4\x97\xf1\x41\x25\x15\xa1\x4a\x7b\xd0\x62\x42\x98\x38\x4d\x4f\xae\x51\xd4\x50\x4c\xa8\x7a\x45\x47\x29\x81\xaa\x09\xab\x95\x1d\x03\x26\x45\x89\x8e\xec\x42\xc9\x68\xfa\x2b\x66\x82\xd0\xcb\x80\x37\x58\x8e\x86\x93\x01\xa7\x47\xd0\x50\x3e\x19\x81\x46\x27\x06\xb2\xd5\x59\xe1\xfa\xfd\xa4\x64\x2f\x48\x89\x36\xa4\xfa\x2d\x39\xe4\xe3\x8b\x86\xb2\x53\x03\x25\x2a\x66\x76\x06\xab\x4a\x02\xc8\xb3\xe6\x67\x5e\x7e\xf0\x03\x5f\xa3\x68\xd3\xf5\xd5\x99\x8a\xfe\x91\xae\x91\xfb\x7e\xbf\xdd\xef\x5f\x0c\x2f\x79\xbe\xcd\x73\x5d\xc0\x42\x6b\x71\x2b\xf9\x1c\x00\xd6\x41\xdc\xc1\x17\xa7\x31\x7c\xb5\xb4\x44\x42\x0a\xea\x0b\xb9\x37\x4d\x41\xda\x55\x0f\x20\xc5\x84\x61\xaf\xea\x68\x3d\x88\x71\xb4\xe1\xaa\x93\xbc\x07\xea\x30\xf7\x64\xea\x9e\xd4\x40\xe6\x74\xf6\xb7\xef\x61\x09\xc2\xa0\xf9\x7b\x11\x48\x17\xdc\x5a\x72\xa9\x4a\xb4\x29\x92\xe1\xe3\x84\x05\xa5\x59\xb0\x62\x47\xdc\x6d\xcb\x31\x73\xb5\x2b\xca\x31\xb0\x0f\x13\x38\xbe\xc5\xe0\xcd\x2e\x9b\x7c\x61\x73\x94\x62\xec\xa9\x12\xa5\x6f\x66\xef\xc4\x73\xd5\x75\x92\x33\x82\xd2\xf6\x82\x36\xc7\x64\xf8\xe8\x64\x3a\x29\x4e\x4f\xb8\x98\xb5\xbd\x0e\xe0\x1e\x95\x65\x55\x66\x16\xd1\x28\xad\x18\x43\xa3\x93\xd9\x82\xc2\xf5\xfb\x2b\x32\x9e\x71\x2a\x4e\xd0\x78\x1f\xfe\xd9\xd1\x16\x2b\x0c\x52\x59\xaf\x29\xc7\x6d\x47\xc9\x93\x2a\x9b\xe9\xa5\x85\x2f\xa6\x59\x88\x9f\xbb\x7d\x58\xab\x4c\x74\x35\x86\x70\x9f\xcf\x20\xe3\x66\xd8\x60\x22\x3f\xcb\x5b\xe8\xd2\xa9\x4a\x3f\x1b\x36\x50\xb3\xbc\xd9\x52\x4d\xb7\xbb\xe4\x3f\xb9\x35\x9d\xa5\xff\xd6\x19\x30\xe0\xf4\x49\xb4\xc7\x78\xdd\x2b\xe9\x56\x04\xa9\xbd\x9d\xd4\x98\xd7\x3f\x86\x4e\x46\x31\x4a\xb3\xa4\xbd\xfc\x7c\x50\xed\x9c\x09\x1a\xcf\x35\x7a\xcf\xe2\x63\xf2\xaf\xf3\xaf\xb0\x1c\xac\x60\xf8\xb5\x62\x8f\xca\x0a\x73\x8f\x2a\xbc\xed\x69\x96\x12\xe6\x55\x7f\x5e\xd9\x78\xbb\x6f\x34\x9e\x05\xb0\x25\x18\x9e\xe5\x64\xcc\x81\xa5\x21\x0d\x6b\xd0\x77\xc1\xb7\x64\xf8\x78\x8e\x6c\x9b\x87\x5f\xaf\x8f\x48\x66\x12\xb9\x8b\x37\x13\xbd\xb2\x93\x7c\x89\xb6\x54\x65\x99\x77\xe8\xf0\x2a\x58\x5f\x14\xb9\x75\x70\x65\xde\x80\x16\x01\xe8\x28\x8a\x36\x75\xaf\x14\x15\x10\x6a\x2c\x67\x10\x48\xcd\xae\xda\x85\x87\x1a\x8d\x41\x9e\xcd\x87\xd3\x02\x67\x68\x74\x8f\x55\x1f\x81\x1b\xe0\x21\x09\x95\xd0\x9c\x34\x6f\x0c\xb2\xbd\xc3\x20\xb4\x73\xdb\x88\xbb\x99\xd4\x34\x09\x25\x2f\x01\xc1\x4b\xd5\xf8\x42\x50\xbc\x39\x04\x08\x05\xcc\xf8\x53\xfc\xd4\x4e\xb2\x56\xab\x6b\x67\x0a\x03\x98\xb3\x09\xd3\xee\x2f\x83\x5d\x2d\x05\x28\xf9\x90\xde\x72\xaa\x25\x0b\x25\x63\x7b\xbf\x46\x91\x31\xa1\xc9\x1e\x86\xf6\x2a\x51\xb2\x7a\xf4\x73\x26\xf4\xfb\x70\x1c\x4f\xf4\xd5\xa9\xc2\x3f\xb2\xdd\xee\x75\xfe\x26\x3f\x03\x52\x26\x32\xec\x3c\x0f\x23\xce\x40\x3d\x8f\xfb\xd3\xf4\x2e\xf0\x87\xfe\xbf\x39\x6f\x3a\xd4\x1c\xf2\x81\xdb\x84\xeb\xca\x0e\x8c\x2b\xd2\xb8\x07\xe8\x03\xf8\x32\x82\x07\xb4\x68\x60\x65\x23\x3f\xee\xb7\x05\x17\xe2\x78\x4c\x8b\x01\xe2\xea\x31\xc5\xf5\x70\x62\x84\x6d\xff\xb8\xfd\xe9\x97\x92\xc1\xd8\xe8\xee\x2c\x75\x23\x85\x70\x0f\xd2\xbf\x37\x69\x6e\x46\x0e\xcb\x25\x4c\xa3\x75\x94\x75\xdf\x79\x5e\x68\x39\xfe\x1f\x8b\xce\xbd\x74\x5d\x5d\x19\xc3\x64\xfe\xa5\xf1\x9b\xa2\x73\x08\xea\xde\xb6\x8a\x7e\xb8\x06\xfb\xb0\x81\xa0\x17\xf8\x8e\xc1\xfa\xc2\x70\x4d\xd7\x17\xc6\x3f\x01\x00\x00\xff\xff\xfb\x2b\x18\xc9\xad\x0f\x00\x00")

func assetsAppCssBytes() ([]byte, error) {
	return bindataRead(
		_assetsAppCss,
		"assets/app.css",
	)
}

func assetsAppCss() (*asset, error) {
	bytes, err := assetsAppCssBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/app.css", size: 4013, mode: os.FileMode(436), modTime: time.Unix(1490535527, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsAppHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x55\x3d\x97\xdc\x2a\x0c\xed\xe7\x57\xf0\xa8\xd7\xcb\x79\xdd\x3b\xef\x60\x37\xf9\x68\x93\x62\x53\xa4\x94\xb1\x76\x86\xac\x0c\x1c\x90\x3d\xeb\x7f\x9f\x83\xb1\x77\x3d\x8e\x27\x49\x35\x70\x75\xb9\xba\x12\xc2\xa3\xff\xf9\xf8\xe5\xc3\xd3\xf7\xaf\x9f\xc4\x85\x7b\x6a\x4e\x7a\xfd\x41\xe8\x9a\x93\xee\x91\x41\x98\x0b\xc4\x84\x5c\xcb\x6f\x4f\x9f\xab\xff\xa4\x5a\x71\x07\x3d\xd6\x72\xb4\x78\x0d\x3e\xb2\x14\xc6\x3b\x46\xc7\xb5\xbc\xda\x8e\x2f\x75\x87\xa3\x35\x58\xcd\x9b\x07\x61\x9d\x65\x0b\x54\x25\x03\x84\xf5\xbf\x0f\xa2\xb7\xce\xf6\x43\xbf\x02\xf2\x56\x15\x42\x20\xac\x7a\xdf\x5a\xc2\xea\x8a\x6d\x05\x21\x54\x06\x02\xb4\x84\x9b\x4c\x13\xa6\xbf\x39\x98\x18\x78\x48\x55\x0b\xb1\x4a\x3c\xdd\x28\xb4\x04\xe6\x25\x6b\x90\x75\x2f\x22\x22\xd5\x72\xa6\xa4\x0b\x22\x4b\xc1\x53\xc0\x5a\x32\xbe\xb2\x32\x29\x49\x71\x89\xf8\x5c\x4b\x05\x29\x21\x27\x05\x21\x3c\x66\xb8\x39\xe9\x64\xa2\x0d\x2c\x52\x34\xb7\xe1\x1f\x49\x36\x5a\x95\x68\x73\xd2\x6c\x99\xb0\x19\x47\xad\xca\xea\xa4\xd5\xd2\xea\xd6\x77\xd3\xd2\x78\x8c\xcd\x49\x08\x21\xb4\x83\x51\xd8\xae\x96\x3d\xba\x41\x16\x6c\xc6\xdb\x81\xd9\x3b\x61\x08\x52\xaa\x87\xb0\xb8\x2c\xa8\x6c\xc8\x26\xd6\xaa\xec\x9a\x5b\x6e\x0b\xe6\x65\xc7\x76\xfe\x2a\x02\xc1\x64\xdd\xf9\xce\xa1\x9c\x7d\x77\x28\x43\x6f\xec\xe2\x55\x39\x18\x77\xb6\xd3\xd0\xce\xce\xc5\xdc\xd1\x5a\x76\x36\xe5\x4c\xff\x0b\xe7\x1d\x6e\xeb\xf1\xf4\xbe\x99\x01\xb2\x3b\x0f\x11\xc9\x43\xb7\x73\x51\xc0\x77\xd7\x8a\xec\x9f\x74\x8c\x77\xcf\xf6\xbc\xd3\x29\xe0\x81\xce\x52\x54\xb9\xa3\x7c\x2b\x3a\xa1\x61\xeb\xdd\x5c\x5e\xee\xf3\x6f\x6b\x7b\xab\x4b\xab\xbc\xd2\x6a\x39\xbd\xd3\x31\x43\x8c\xe8\x78\x3d\x64\xd8\xe2\xe2\x56\xce\x43\x92\xef\x48\xcc\x2b\xad\x72\x70\xe1\xa5\x00\x6b\x55\x12\x22\x67\x33\x99\x58\x96\x5a\xe5\xf0\x01\x13\x09\x42\xc2\x6e\xa6\xbe\x5d\xfb\x86\x9b\xef\x6e\xa1\xe6\x37\x12\x3d\xdd\x1d\xbc\x88\x01\x81\x1b\x6d\xfb\xf3\xed\xdc\x97\xc0\x63\x1a\xcf\x52\x00\x71\x2d\x0b\x90\x1f\xc2\x76\x64\x0e\x14\xc1\x75\xbe\x3f\x52\x9c\x03\x5b\xc5\x19\xd8\x2b\x6e\x87\x70\xa0\xb5\x8e\x0e\x19\x2c\x65\xee\xb0\xbb\x85\x67\xef\xf9\x97\xc7\x96\xbb\xd2\x96\x8f\xc2\xb1\xcb\x10\x71\xbc\x9d\x20\x31\x02\x0d\x58\xcb\x1c\x91\x07\xf6\x33\xbe\x31\xbf\xd0\x8e\x5f\x5b\x4e\x7f\x47\x9d\x60\x52\x01\x86\x84\x87\x39\x08\xa6\x6d\x0e\x82\xe9\x6e\x0e\x87\xaf\x7c\x9c\x23\x47\x8e\xd4\x33\xbe\x51\x5f\x68\x47\xcd\xd7\x6a\xed\xaa\x56\xe5\x9b\x96\xdf\xcf\xfc\xaf\xf2\x33\x00\x00\xff\xff\x90\xf1\xb6\x02\x6d\x06\x00\x00")

func assetsAppHtmlBytes() ([]byte, error) {
	return bindataRead(
		_assetsAppHtml,
		"assets/app.html",
	)
}

func assetsAppHtml() (*asset, error) {
	bytes, err := assetsAppHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/app.html", size: 1645, mode: os.FileMode(436), modTime: time.Unix(1490535256, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsAppJs = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x3c\x5d\x73\xdb\x38\x92\xcf\x9b\x5f\x81\x61\xd5\xad\xa8\x33\x23\xdb\x73\x77\xfb\x40\x8f\x26\xb5\x3b\xc9\xed\xcc\x55\x32\xd9\x9a\xf8\xe1\xaa\x54\x2a\x17\x24\xc2\x36\xc6\x14\xc9\x01\x20\x39\x2a\x47\xff\x7d\xab\xf1\x41\x82\xf8\xa0\x68\xc7\xce\xf8\x81\x22\x89\xee\x46\xa3\xd1\xdd\xe8\x6e\x80\xde\x61\x86\x76\x3b\x34\x87\xcb\x97\x2f\xe8\xe1\x15\x42\x08\xd5\xab\xdf\x73\xf4\x70\xc8\xe4\x03\xaf\xab\x9b\xfe\x13\xb7\x1e\x45\xcd\xf0\x0d\xe9\x5e\x6c\xea\x82\x94\x39\x7a\x28\x29\x17\xf0\x56\xbf\xde\x51\x72\x9f\xa3\x87\x0d\xa6\x95\x84\x45\xa6\x3d\x43\x1b\x52\x6d\xd5\x5d\x53\xe2\xfd\x0a\xaf\xef\xd4\x13\x29\x71\xc3\x49\xa1\x1e\x0a\x56\x37\x45\x7d\x5f\x59\x24\xd7\x75\x25\x58\x5d\x22\xd5\xf7\xe1\xe2\xd5\x6e\x37\xab\x57\xbf\xa3\x39\x4a\xaf\xb7\xd5\x5a\xd0\xba\x4a\xa7\x6a\x3c\xe6\x19\xdd\x10\xf1\x91\xbd\x2b\x39\x49\x37\x19\xba\xcb\xd0\x6e\xaa\x47\x0c\x7f\x8c\x88\x2d\xab\xd0\x1d\xa2\x15\xda\xbc\x41\x9b\xc5\xdd\x12\xe5\x68\x77\x21\x01\x0e\xaf\x2c\x90\x87\x96\x4e\xde\x91\x3c\x5c\xbc\x3a\x4c\xd3\xa9\xe4\x03\xa4\x14\x62\x04\xa4\x2d\x30\x34\xb5\x2d\x00\x9a\xa1\x3b\xb2\xe7\x19\xaa\xc5\x2d\x61\x36\x4b\xd7\x35\x43\x29\x05\x86\x00\xc0\x6e\x31\xd4\xee\xc8\x1e\xcd\x65\xeb\x82\x2e\x2f\x7a\xcd\xf4\x1a\xa5\xd0\x4c\x2b\x39\x6b\x2e\xb6\x35\x1e\x68\x5e\xdc\x91\xbd\x43\xe0\xf0\xca\xbf\xd3\x18\x92\x53\x5b\x32\xc0\x4b\x2b\x8a\xe0\xf8\xfc\xe1\x0d\x33\x18\xe5\xed\x80\x08\xf4\x61\xb0\xe7\x73\x94\xfc\xbd\x5c\x6d\x37\x9f\x6a\x26\x92\x30\x11\x81\x6f\x34\x23\x0b\x05\x9b\x2c\x0d\x3b\x83\x64\x99\xa0\x5c\x8c\xa5\x2b\x81\x47\x12\x06\x1e\x34\xc2\x8b\x51\x7e\x8c\x3c\x74\x17\x19\x7a\x7c\x67\xcf\x23\xf6\x10\xba\xa5\x64\xad\x9e\xf5\xb4\x2d\xa8\x67\x01\x93\xee\xcc\xde\xd2\xc6\xc9\xa2\xaa\xd1\x04\x9d\x48\x13\x3a\x41\x93\xe5\x64\xea\x2a\x34\x17\x2c\x6c\xaa\x76\x1f\x12\xb0\x66\x42\x59\xe2\x64\x72\x31\xd2\x7a\x0d\xce\xc9\x3c\xc4\x1f\x98\x73\x86\x26\x68\x32\x8d\x1b\xa1\xa6\x10\x72\x50\x2d\x68\xc8\x53\x65\x76\xab\x7c\xdf\xbd\xe1\x82\xe5\x70\x51\x6f\xc0\xa7\xa5\xd3\xce\xa7\xf1\x98\x53\x03\x56\x5c\x51\x71\x5f\x56\x96\x4d\xf3\xd9\x06\x37\x69\x0f\xc1\x95\x90\x86\x5e\x28\xa1\x68\x16\x66\x5c\x30\x7b\x26\x96\x9d\x7c\xa6\x33\x60\xa3\xa5\x89\x52\x9c\xa1\x95\x4b\x14\xf4\x17\x2f\xce\x97\xe8\x07\xb4\x5a\x9c\x2f\xa7\xe8\xc1\xf4\xf3\xfa\xfc\xa2\xd5\x46\xf3\x0e\x5e\x5d\xd8\x3d\xf4\x99\xb6\xb0\xf9\xe2\x6c\x79\x81\x0e\x9e\x0e\x6d\x2b\xfa\x47\x58\x32\x51\xc1\x5c\xd3\x52\x10\x66\x8d\x43\x0d\x97\xa2\x0c\x71\x52\x5e\x87\x46\x44\xc1\x1e\xcf\x06\x5c\xbc\x60\x5b\xe2\x78\xf7\xce\x9e\x8d\x68\xc3\x76\x82\x26\x53\xf4\xdd\x1c\x05\x80\x48\x79\xbd\xa0\xe8\x35\x3a\x5f\x5a\xb0\x4f\x61\x22\x8a\x71\x8d\x4b\xee\xa2\x58\xd3\xe1\x0a\x5b\x49\x2e\x20\x6e\xd5\x10\xd7\x45\x47\xe4\x46\xe2\x61\x81\x4b\xbb\xbe\x06\xbb\x0e\x90\xb5\x67\xc5\x12\x99\xa6\x78\x2d\x45\xa9\xd1\x16\xd7\xcb\x10\xe6\xf0\xf0\xfb\x22\xf0\x9f\x82\xa2\xee\x4b\xca\x73\x12\x60\x36\xb9\xbc\x76\x8e\x00\xf4\x36\x97\xd7\xee\x9d\xe2\x3b\xd7\xbf\x01\x17\xa1\xa2\xc1\x68\xe4\xc3\x08\xb4\x2d\xf4\x52\x0e\xaf\xd6\x5b\xc6\x48\x25\xc2\x6f\xaf\x4a\xcc\xc5\xd5\xa6\x2e\xe8\x35\x25\x05\x9a\xa3\x24\xb1\x60\x74\xf8\xe7\x60\xaa\xb7\x83\x98\x25\x5d\x31\xcc\xc0\x59\x77\x12\xe8\xad\x82\x39\x5a\x2c\xbb\x41\xa3\xe4\x9f\xa4\x62\xc4\x7a\x7b\x70\x49\x05\xbb\xeb\x7c\x23\xde\xf5\x22\x22\x7b\xce\x05\xdb\x3b\x1a\x50\xd6\x6b\x5c\x7e\x52\x82\x9c\x69\x89\xfd\xdf\xa7\x8f\xbf\x82\xdf\xa3\xd5\x0d\xbd\xde\xa7\xf0\x76\x6a\x4d\x2e\x5a\x63\xb1\xbe\x45\x29\x99\xa2\x07\x77\x95\x2c\x6b\x5c\x8c\xef\x1c\x94\xd6\x63\x20\xa4\xa3\x36\x63\x0d\x66\x9c\x04\xd0\xa2\x66\x1b\xe3\x17\x78\x4d\x35\x9a\xa7\xa3\x40\x32\x97\xd7\x6e\x6a\xb4\x9e\xe4\xe6\xc6\x6b\xe9\x4f\x4c\x1e\x7e\x6d\x61\x29\xed\xc9\xcd\x8d\xd7\xe2\xd1\x0b\xbd\xee\xb0\xb4\x7a\xe4\xe6\xc6\x6b\x71\xe9\x05\x5f\x5b\x8b\x33\xde\x91\x5c\x5e\x2d\x4a\x35\x06\xc4\x1a\x17\x8e\x3d\xbe\x02\x8b\x94\xe9\xd8\x0c\x92\xad\xbe\x51\xa2\xce\x2a\x2f\x7f\x7b\xf7\x6e\xc8\x16\xfa\x93\x9f\x80\x97\x48\x72\x4f\x21\xbc\x38\xf2\x2d\x16\x44\xc6\x93\x32\xee\x83\x17\x94\xaf\x7f\xdd\x6e\x56\x84\xc1\xd3\x25\xc3\xeb\x3b\xeb\x91\x8a\x52\x82\x5f\xd3\x92\x24\xb6\xf5\x41\x9f\x30\xeb\xa1\x3e\xbd\x4e\x9b\x12\xd3\xca\x45\xb7\xf9\x03\x20\xac\x23\xd1\x10\x50\xcb\x07\xb8\xeb\x64\xe9\x81\x58\x58\x87\xee\xb6\xf5\x10\x63\x65\xa5\xe0\x5f\x5c\x3a\x6d\x37\xcf\x2d\x17\x1f\x66\x19\x94\x8b\x54\x82\xf1\x62\xf9\x36\x3a\x63\x7a\xf9\xa6\x42\x71\xa3\xc2\xa6\xc0\xa2\xb7\x2a\x14\x58\x60\x37\xad\x68\x83\x7b\x64\xa2\x0e\x9d\x2e\x83\xcd\x7a\xd5\x80\x76\xfd\x9d\x69\x37\x22\xd3\x66\xd4\x46\x6e\x5c\x85\xc7\xd0\x51\x26\x29\xc8\xf6\x85\x9a\x8e\xa5\xbd\xa0\x18\x47\x62\xf8\x60\x75\x2d\x2a\xbc\x89\xae\x62\x12\x06\x96\x3d\x00\x4c\x3a\x42\x26\x06\xb2\x56\x85\x59\x49\xaa\x1b\x71\x0b\x51\x90\x17\xae\x32\xcd\xab\x05\xbe\x38\x5b\x2e\xce\x97\x2e\x6b\xa8\x5b\x20\x4c\x11\xe2\xa2\x97\x8a\x40\x66\x70\x8c\x59\x33\xa8\x74\xda\x67\x98\xc9\xd4\x56\x8e\x24\x96\x90\x0c\x31\x24\x05\xcb\x5a\xb1\x7a\xe9\x40\x13\xe3\x4b\xf6\xdd\xf2\x04\x02\x0a\x33\xe1\x0a\xb4\xa9\x9b\xd4\x59\x6b\x2d\x10\x58\x2c\xd2\xa1\xb9\x2d\xea\xfb\xca\x66\x69\x87\xcb\x2d\x19\x2b\xaf\xae\x02\x15\x9a\x7a\x16\x1f\x84\x42\x6a\x65\x25\x8d\x75\xb9\x88\xe8\x0a\x24\x18\x90\x5d\x05\x84\xee\x09\x63\xcb\x6f\xd3\x85\xcc\x45\xe4\x40\x6c\xb5\x8e\x48\xc5\x92\x05\x5e\x71\x37\x77\x70\x25\x41\x33\x29\x08\x9d\xef\x70\x52\x92\xb5\x20\xc5\xc5\xd7\xce\x22\x84\xb1\xbe\xea\x3b\xf5\x38\x20\x74\xdc\x44\xe0\xcf\xf0\xd5\xca\xb8\xae\x45\x2b\xe6\x0b\x3f\x99\x91\x45\x0a\x83\x14\xcb\x66\x64\x8e\x69\x80\xac\xb9\x89\xe5\x30\x2b\x46\xf0\xdd\xb1\xe4\xa5\x53\x06\x43\x79\x41\xfb\x93\x3d\x66\xaa\xbd\x3c\x0b\xd2\xeb\x65\x34\x02\x45\x8f\xb4\x11\x1d\x3f\x45\xdd\x49\xc9\xad\x34\x04\xf9\x5a\x70\xcc\xa1\x00\xfd\x2b\x00\xe8\x71\x11\x4a\x8d\x6d\x84\xf5\x2d\x2d\x8b\xa3\x7c\x2b\xb0\x41\x67\xa8\xd4\xaa\xe3\xb7\xbf\x90\x75\xe9\x52\x60\x85\x91\x8a\xd5\x87\xd7\xc9\x2d\x84\x94\x87\x2c\x6a\xf4\xb6\x42\x1e\xb1\xfb\x3e\x11\x2e\xf6\x25\x79\x3c\x99\x73\x97\x8c\xaa\xd0\x3f\x1c\xfa\x3a\x22\xf6\x0d\x10\x4f\x0a\xca\xc6\xad\x63\xf3\x10\x27\xba\xd1\x9d\x6d\x43\x5c\xc6\x2b\x21\x6f\x26\x6d\xb1\x24\x44\xd6\x16\x9c\x0e\x43\x55\x1f\x09\xaa\x0b\x3f\x32\x17\xa1\xd5\x96\x5c\x38\x8a\x6e\x8a\x0d\xae\xdb\x00\x64\x10\xef\x32\xe0\x52\x54\x5b\x78\xe1\xed\xeb\x83\x5d\x3d\x31\x59\x4e\x5b\x16\xb9\x18\x42\xda\x56\xf4\x8f\x0e\x05\x0c\xf6\xc2\x5d\x4b\x95\x75\xb7\x30\x72\xea\x33\x29\xc6\xa5\xe7\xbb\x5b\x1b\x1a\x54\x75\x22\x1c\x4b\x75\xc2\x9b\x24\x71\x02\xae\xb6\x2d\x12\x75\x31\x22\x94\x27\x7a\x50\x16\x9e\xb7\xd4\x0e\x9e\x65\xda\x23\x53\xd0\x19\x3c\xb7\xa1\x68\xa6\xf4\x6e\x39\x58\xab\x51\x91\x63\xae\x7f\x3b\xa5\x36\xdd\x76\x0c\x58\x79\xa3\x0e\x89\xf2\xf6\xce\xaa\xf3\x34\x40\xab\x7b\x56\xbb\x6c\x70\xed\xde\xe1\x15\xcf\xe1\x62\xe7\xaf\x5c\xe4\xf2\xea\x57\x81\x76\x94\xdc\xcf\x36\x98\x56\xd1\x62\xf1\x6d\x7d\x3f\x34\x4b\x30\x11\x45\xbd\xde\x6e\x48\x25\xc0\xa7\xbf\x2b\x09\xdc\xfe\x63\xff\x4b\x91\x26\x3a\x85\x4f\x2c\xe9\x92\x99\xd4\x8c\x59\x41\x79\x53\x62\x19\x8d\xac\xca\x7a\x7d\x97\x78\x4a\x72\x4b\x8b\xc1\x20\xf6\x19\x7a\xae\xea\xca\x18\xf7\xc1\xee\xb7\x20\x55\xac\x67\x3d\xc7\xc7\x3b\x76\x7b\x8b\x75\xe7\x67\x17\x5f\x3b\x50\x2f\x1b\x91\xdd\x94\x68\x8e\x88\x85\xce\xff\xb1\xff\xa9\xc4\x9c\xff\x0a\xcb\x48\x52\x10\x81\x69\x99\x4c\x7b\xab\x39\xe0\x55\xe4\x5e\xa2\xb6\x0c\xac\x19\xc1\x82\xbc\xd5\x8f\xff\xcb\xf0\x0d\xfc\xba\x81\x66\x49\x6d\xc1\xc7\x7a\x15\x32\x29\x83\x4e\x67\x82\x7c\x16\x3f\xd5\x95\x50\x05\x46\xcb\xc7\xe9\x01\x9a\x0c\x6e\x39\x86\x2e\xd6\xbb\x75\xe3\x08\x9b\x5d\xb4\x8e\xf2\xb6\x9c\xd1\xaa\x22\xec\xe7\xcb\x0f\xef\x43\x8e\x46\x67\x76\x3e\xb1\x90\xe3\x37\x1b\x70\x8a\x7f\xf4\xe5\x0b\xea\x6f\x59\x7a\x81\x06\xd2\x35\x2c\xb9\x40\x0c\x44\x45\x25\xf5\xe7\x45\x8b\x23\x4d\x4a\x9a\x38\x21\x55\x49\x1d\x59\xa8\xbd\xb4\x24\x47\x09\x3a\x09\x09\xc6\xdf\x64\x96\xca\x30\xc3\x4d\x43\xaa\xe2\x27\x19\xd1\x94\x34\xe8\x39\x1d\x28\x89\xd7\x0f\xe1\xfd\xba\xf6\x6d\x7d\x9f\xcb\x6b\xe7\xb9\xc0\x05\xe4\xf2\xda\x7b\x57\x90\x2a\xd7\xbf\xd9\xa0\xaf\xf5\x3c\x9d\x5f\x5c\x7b\x26\x4f\x57\xca\x89\xfc\xe6\x6e\x6e\x44\xb7\x2f\xe0\xe3\x54\xaf\xcf\xe7\xe0\x64\x48\xde\x2b\x80\xca\x8b\x1b\xdc\x9a\x4d\xcd\x92\x7b\x11\xa2\x8e\xd6\x4a\xbe\xf8\xaf\x58\xec\x98\xf9\x9a\x3c\xc6\xad\xf5\xd1\xfa\x38\x11\xb9\xc8\x20\x9e\x91\xca\x0b\x87\x6d\xa7\x78\xcc\xc5\x50\x73\xd2\x22\xb0\x17\x2d\x47\x24\xdb\xbc\x73\x24\xd2\x27\x6c\xf0\x1d\xb9\x92\x31\x16\x15\x64\x93\x1a\xc8\x0c\xe4\x73\xa6\x7e\xbe\x5f\xaa\xd8\xcc\xf7\x11\xb8\x28\xde\xed\x48\x25\xde\x53\x2e\x48\x45\x58\x3a\x59\x97\x74\x7d\x37\xc9\xc2\xf3\x67\xfe\xc0\xd3\x7d\x67\xcc\xcc\x9c\x00\x9a\x29\x3d\x4b\x83\xfb\x8c\x48\x25\x76\x1e\x02\x71\x2b\x24\xe6\x4f\xa9\xe5\x98\x24\x55\x1e\x94\xc2\xe5\x16\x74\x42\xdc\x52\x0e\x33\xf5\x77\x21\x18\x5d\x6d\x05\x49\x93\x3b\xb2\x4f\xa6\x81\x2a\x21\x4c\x0b\xa3\x61\x9c\x2d\xf3\x1c\xaa\x19\xb6\x52\x3e\x9d\x88\x0c\x8c\xd4\x52\x6f\x18\xab\xae\xde\x84\xc7\x6a\x3b\xac\x99\xb2\xa0\x90\x58\xa2\x5b\xb2\x9a\x86\xde\xf3\x98\x81\x85\xa6\x5b\x46\x43\x24\xfa\x2b\x4c\xa6\xf6\x31\xa7\x4f\x74\xfc\x8f\xb0\x8d\x63\x2b\x04\x4c\x62\x5f\x93\x23\x47\x95\xac\x54\xc3\xf3\x2c\xe3\x97\x48\x59\x30\x02\x8b\x74\xac\xb1\xa4\x33\xde\xd3\x84\x35\xc4\x19\x89\xee\x76\x3a\x00\x08\x6a\x16\xab\x77\x0c\xe1\x81\xaa\x65\xea\x2c\x95\x2e\x97\x3b\x55\x4f\x9d\x56\xcf\x75\x31\xdb\x0b\x3a\xcc\x28\x7e\xe0\x0d\xae\x90\xe4\x77\x2e\x18\x5e\xdf\xfd\x98\x9c\xf8\xec\xf4\x4a\xf5\xd3\x93\xe4\x87\x53\x40\xfb\x31\x39\x09\x2a\x55\x9f\x28\x84\x34\x11\xa2\x2a\xb4\xeb\xc8\xf9\xe7\xdd\x02\x58\x6d\x44\xd4\x3f\xc3\xd0\x36\xdb\x47\xb1\x82\x5e\x48\x0e\xfd\xc4\x19\xbb\x0a\x07\xc3\x7c\x1a\x6a\x31\x46\x0f\x63\xd8\x96\xb3\x24\x4b\x47\x76\xa3\x1f\x52\xb5\xa0\xe3\x79\xd7\x27\x2b\xc3\xcc\xbf\x97\xa5\x8b\xe3\x93\xe6\xcf\x9c\x2a\x7a\x5c\x71\xd2\x60\x86\x45\xcd\x7e\x3c\x1d\x35\xfc\x20\x8b\x8a\xd6\x58\x0e\xbd\xa2\x59\x5f\xa3\xd5\x1e\x4e\x58\xa5\xdd\x8e\xc1\x2b\x86\xbb\x95\xdb\x46\x71\xe5\x0b\x6b\x09\xf4\x1c\x51\x12\xc5\xd4\x13\xe8\x0d\xaa\x9e\xad\xcd\x03\x02\x0a\x5b\x77\xd8\xaf\x84\x8b\x40\xbd\x50\x43\xde\x7b\xe5\x1b\x13\x9b\xfc\xa9\xe1\xf9\x86\x54\xdb\xa1\xf0\xfc\x8a\x6f\x57\x4f\x8e\x95\xf9\x76\x05\xf4\x1f\x13\xa5\xf7\x82\xf4\x6f\xd0\xf9\x40\xac\x3e\xd4\xfb\xb1\x78\xbd\xed\xfc\xf9\x42\x76\xb9\x35\x16\xed\x50\xf7\x16\x4b\xcf\xb7\x4d\xa0\xd0\x50\xe2\x15\x29\x95\x5e\x77\x91\x8f\x89\x1f\xdf\x20\x15\x40\xa0\x1c\x01\x76\x7f\x39\xdc\x36\xbd\xc4\xf6\xbb\xb9\xa2\xe5\xfa\x10\x07\x4c\x43\x79\x15\xf9\xb8\xfa\xc3\x14\xe4\xed\x5d\xdf\x0c\x54\x9b\xb9\x73\xcd\xa1\x6d\xd5\xf7\x8f\x34\x0b\x73\xd6\x3e\x66\x1a\xc3\xd3\xe5\x14\xc4\x75\x48\xb8\x48\xb8\x00\x27\xb9\x94\x5a\x00\x3d\x78\x4e\x37\x3a\xbd\x86\x9f\x81\x29\x56\x04\x21\xca\xb3\x23\x3e\xce\xd6\xa0\xe6\xa7\x98\x73\x22\xf8\x69\x83\xb7\x9c\xcc\xf8\xee\xe6\x98\xcf\x7b\x79\x4e\x4a\xbc\x77\x19\xe9\x69\x67\x77\xf6\xed\x31\x95\xb8\x98\xe8\x19\x69\x08\x16\x89\x77\xa2\x50\x93\x88\x8e\x45\xe3\xf9\xa3\x91\x66\x5d\x37\x78\x4d\xc5\x7e\x7e\x3e\x3b\x3b\x22\xcf\xe7\xe9\xe7\x6c\xf6\x3f\x21\x71\x45\x07\x8d\xab\xa2\xde\x3c\x61\xd0\x0a\xef\xe5\x07\x3d\xaa\x9f\xc0\xa0\x1f\x51\xfc\x3f\x38\x96\xad\xa3\xbb\xe8\x79\xaf\x61\xcb\x2e\xb0\xc0\x4e\x4d\x53\x09\xbb\xaf\x81\x13\x69\xe9\x13\x44\x2b\xe4\x1e\x5c\x31\xfd\x74\x7c\xc8\x03\x82\xbf\x54\xea\xe8\xc9\x42\x26\x18\x57\xba\x35\x59\xa2\xff\x44\xe7\x67\x67\x67\xee\x09\x86\x9e\x7d\x68\x60\x1f\xc4\x3d\x77\xe9\xf4\xd4\x6b\x1e\xea\x4a\x4b\xa4\x22\xf7\xe8\x6d\x20\x2d\x86\x11\x6b\xde\x8f\x39\x38\x4b\x2b\x20\x7a\x03\xc2\xa0\x1d\x97\x54\x6e\x04\xbf\xee\x73\x3c\x10\x16\x77\x83\x6f\xc7\x64\x5e\x9d\xc6\x46\xb1\x91\x5b\x2e\x01\xf8\xbf\x9d\x4d\x3d\x60\x4e\xc0\x5b\x19\x98\xff\x40\x7f\x3b\x0b\x48\x57\x2d\x9f\x40\xf7\x04\x4d\xf2\x09\x3a\x41\x69\x72\x96\xa0\x13\xc0\x9e\xce\x78\x49\xd7\x24\x7d\xfd\xbd\x4f\x1c\x16\x45\x1e\x76\x6c\x7d\x03\x31\x6a\x10\x18\x0d\x8d\x9d\x53\x90\xc4\x63\xf5\x22\xd9\xb8\xa0\xcb\x51\xcb\xb7\xf9\x0b\x22\x79\x8b\x79\x78\xa2\xdc\x8f\xa4\xfa\x96\xdb\x37\x58\x6f\x15\x36\x35\xaa\xa8\xb1\x8e\x2a\xaa\xda\xa1\xee\xac\x8b\x08\x52\xef\x78\xfc\x60\x5d\xd8\x21\x42\x7c\x12\x9e\x3b\x8a\x05\xe6\x6e\x10\x6f\xb9\xa8\xee\xd8\x76\x78\xc0\x37\x44\x5c\x31\xf2\xc7\x96\xf4\x4f\x5c\x34\x58\xdc\x66\x88\x5e\xb7\x67\x62\xd1\x1a\x97\x25\x2c\xd3\x6e\x14\xf9\xf9\x96\x69\x4b\xfe\xff\x0f\xef\x7f\x16\xa2\xf9\x4d\x51\xb3\x6d\xfa\xf3\x2d\x9b\xd5\x15\x23\xb8\xd8\x4b\x83\x5e\xdf\xe2\xea\x26\x2a\x19\xa3\x5a\x80\x25\x71\x3e\x09\xe9\x2e\xe6\xe8\xbf\x63\x5a\x08\xa0\x40\x79\xcb\x01\xec\xfb\xb3\x33\xf4\xd7\xbf\x06\x39\xb6\xff\x4c\x7b\x6a\x9d\xaa\x56\x7d\xf2\xa6\xae\x38\xb9\x24\x9f\xc5\x34\x93\xcc\xdf\x10\xf1\x9b\x7e\xfb\x33\xc1\x05\x61\x69\xf2\x1e\x73\xf1\xfa\x83\xf1\x75\xd3\xe3\x55\xb9\x4e\x6f\x1d\xc1\x34\xa4\x4a\x93\x7f\xbe\xbb\x4c\x32\xa4\xc4\x2e\x58\xaf\xa6\x28\xcf\x02\xb5\x33\x21\x8f\x38\x79\x6e\x50\x0a\x00\x98\x94\xb2\x37\x3c\xfe\x72\xdd\x72\xf8\xfa\x13\xad\xd6\x24\xb1\xe7\x34\x98\x68\x2a\x42\x55\xe1\x2b\xb3\x32\xa9\x2b\x5d\xba\x0e\xce\x9c\xa5\x4c\x69\x82\x1b\x7a\x2a\xcb\xd6\xa7\x26\xaa\xca\x02\x3b\x54\xce\xd1\xeb\x8e\xae\xdc\xa8\x6f\x59\x0d\xa8\x07\x23\x62\x91\x10\xc6\x6a\xc6\xd5\x02\x51\x6d\xcb\xa0\xc7\xf1\x3b\x45\x73\x24\xb1\x61\x8d\x71\xcf\x67\x85\x31\xbc\x2f\x0e\xcc\x6d\x10\xb9\xdd\x91\x1f\x28\xfb\xea\x10\xcb\x2a\x28\x8f\x39\xc6\x66\x75\x62\x61\xe2\x15\x4f\xad\x01\x1d\xd5\x45\xf8\x3b\x3d\x35\x91\x89\x09\x1c\x04\xbe\x89\x0e\x66\xa0\x86\x1d\xf8\x3e\xc8\xfa\x18\xc3\x68\x8d\x36\xce\x91\x7a\xa3\xbd\x96\xa3\x31\xc1\xc3\xff\x2f\xad\x31\xad\xff\x1c\xad\x31\x91\x4f\x62\x8e\x6b\x8c\x9e\x88\x01\xa5\x71\xb3\xc9\xd1\x53\x72\xbc\x9b\xc0\xa4\x75\x87\x88\xc6\xcd\x9a\x86\xef\xcf\x5a\xf8\x13\x8b\x17\x9a\x35\xcb\x26\xf4\xe8\x86\xcd\xe2\x18\x9f\xd1\x69\x1b\x56\xfa\x86\x91\xdd\x63\x75\xfd\x0d\x96\xb0\x73\xc0\x4d\x32\x70\xf1\xc6\x01\x77\x64\x4b\xbc\xbf\x92\x09\xf7\x50\x59\x87\xab\xd5\x12\xa9\xcf\xf3\xad\xcf\x07\x7d\x1d\xcd\x90\x0e\xb0\xe5\x4d\xdd\x34\xfd\xc8\x50\x1e\xd1\x55\x9f\x44\xce\x0d\x59\x13\x87\xa3\x37\x28\x91\xac\xc8\x7a\x8e\x7c\x75\x31\x7a\x8c\xc9\x89\xba\x09\x0f\xb3\x22\x9f\xa3\x87\xca\x8e\x51\x06\xdc\x01\xe9\xd9\x64\xb7\x8c\xbe\x58\x2c\x13\x5c\xe2\xff\xf5\xf1\xd3\xa5\xfc\xc4\xc1\x2c\x8c\x89\xb7\xd4\x87\x97\x71\x1d\x21\xbf\xbe\xdc\x37\x44\x11\x68\x4a\xba\xc6\xd0\xd9\xe9\xef\xbc\xae\x12\x8f\x42\x55\xa4\xce\x37\x6c\x3d\xed\x7d\x48\x94\xb8\x92\x5c\x7f\x1c\xe2\x6c\xa4\x26\x77\x64\xcf\x93\xdc\xb1\x28\x73\x96\xcd\xdd\x50\x47\x72\xb7\x2b\x47\x5b\x46\x23\x36\x32\xf5\x67\x83\x56\x34\x3a\xc9\x6d\x16\xb3\xaa\x8b\xfd\x98\xcd\x6c\xef\xb0\xe6\x60\x70\x8d\x5c\xbf\x08\xa9\x9c\xaa\x5a\x1f\xa9\x86\x76\x38\x92\xf0\x91\xda\xe8\x93\x38\xd7\x61\x41\x17\x41\x0c\x6d\xbf\xbb\x1e\xcf\x5b\x07\x62\x5b\xcb\x7e\xf8\x10\x38\x07\x35\xb4\x8b\xe4\xb2\xe8\x2d\x41\x23\x02\x87\x1e\x08\xbf\xad\xef\x63\x00\x52\xd8\x61\x1a\x04\xb8\x6e\xfe\xc5\xea\x06\xdf\x60\xa5\x48\xe1\x49\x1e\x9c\x30\x55\x7e\x7c\xea\x94\xb9\xd5\xee\xe8\x40\xa5\xb0\xfe\xd4\x81\x2a\x35\x7e\x16\xdd\x74\x73\xdf\x88\x7e\x76\xc0\xa6\xe8\xfe\x18\x25\x1d\x36\x62\xe4\xa9\xe5\x68\x29\xc9\x85\x52\x6d\xa9\x3c\x72\xcb\x47\xbf\x1a\xa8\xbb\x96\x35\x2e\x9e\x2e\xe4\xb2\x56\x9e\x7d\xa6\x08\x0d\x0d\xc0\xda\x53\x18\x51\x5d\xef\x70\xdb\xe8\x31\x5a\x6f\x87\x18\xe4\x6b\xec\xa1\x3d\xb3\xc2\xc8\xee\xc9\x9a\x7c\x9c\xcd\x76\x5b\xe0\x6b\xd9\x6c\x03\xab\x97\x63\x56\x46\x26\xcf\xc1\x2c\x10\x7a\x34\x9b\x7d\xbd\xa9\xcb\x92\xc6\xb3\x78\xa7\x3f\x2b\xef\x0f\xf8\x2d\x17\x4a\xe6\x79\xc7\xe1\x74\xb4\x0d\x80\x7f\x01\x88\xbf\x70\x55\xb3\xad\xb7\x22\xd5\xec\x65\x6e\xd9\xf5\xa0\x40\x75\x73\x28\x6f\xe1\x02\xb3\x68\x54\x21\x8b\xca\xc6\x4c\xac\xba\xd2\x77\xf3\x39\x9a\x80\xa9\xd1\xea\x66\xe2\x1f\x5a\xa0\xc7\x3f\x12\x6a\xa9\xfa\x33\xfb\xf6\xe3\x07\x1d\xbd\xbd\xaf\x71\x41\x8a\x49\x26\x49\x86\x3e\x22\x92\xbf\x91\x1d\x88\x2b\xf5\x4f\xc1\xac\x07\x77\xfb\x51\x4b\x3e\xef\x3f\x7a\x50\xed\x17\xf2\xfd\xe7\x0e\x0e\x2c\x36\x97\xd7\xac\xa7\xdb\xca\x3c\x72\xeb\xbe\x6b\x07\x8d\xcc\xe5\xb5\x8f\xa3\xa0\xed\xff\x75\x83\xe5\xff\xbd\x80\x1f\xf7\xe3\x79\x4b\x45\x64\x3b\x08\xfd\xdf\x01\x00\x00\xff\xff\x3b\x03\xab\xe5\x2c\x4d\x00\x00")

func assetsAppJsBytes() ([]byte, error) {
	return bindataRead(
		_assetsAppJs,
		"assets/app.js",
	)
}

func assetsAppJs() (*asset, error) {
	bytes, err := assetsAppJsBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/app.js", size: 19756, mode: os.FileMode(436), modTime: time.Unix(1490535939, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsNextSvg = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x64\xcc\xc1\x6a\x84\x30\x10\x06\xe0\xbb\x4f\xf1\x33\xbd\x5a\x33\x49\x55\xb0\x24\x1e\x7a\xef\x43\x14\xb4\x49\xa8\x5d\xc5\x84\x44\xf6\xe9\x17\x57\x61\x17\x96\x61\x06\xfe\xe1\xe3\xd7\x21\x59\x6c\xff\xd3\x25\x18\x72\x31\x2e\x9f\x42\xe4\x9c\xab\xfc\x51\xcd\xab\x15\x8a\x99\x45\x48\x96\x90\xfd\x10\x9d\x21\xc5\x4c\x70\xa3\xb7\x2e\x9e\x21\xf9\x31\x7f\xcd\x9b\x21\x06\x43\xf1\x7d\xa9\x2f\xb4\x45\x88\xeb\xfc\x37\x1a\x7a\xab\x79\x1f\x3a\x1f\xef\x67\x95\x54\x84\x5f\x3f\x4d\x0f\xd0\x17\x00\xa0\x97\x9f\xe8\x30\x18\xfa\x46\xdd\x94\x2d\x43\xca\xa6\x94\xcc\x7b\x92\x35\xe3\x4a\xe2\x05\x76\x07\x6c\x0f\xd8\x3d\x41\x2d\xec\x7e\x42\xb2\x7d\x71\x0b\x00\x00\xff\xff\x22\xf8\x97\x68\xec\x00\x00\x00")

func assetsNextSvgBytes() ([]byte, error) {
	return bindataRead(
		_assetsNextSvg,
		"assets/next.svg",
	)
}

func assetsNextSvg() (*asset, error) {
	bytes, err := assetsNextSvgBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/next.svg", size: 236, mode: os.FileMode(436), modTime: time.Unix(1490506165, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsPauseSvg = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x8e\xe1\x4a\x03\x31\x10\x84\xff\xdf\x53\x0c\xeb\xdf\xda\xec\xc6\xd6\x2b\x72\x39\xd0\xff\x3e\x84\xd0\x73\x73\x78\xde\x95\x26\x24\xc5\xa7\x97\x94\xd4\x62\x95\x90\x5d\x32\x7c\x99\x99\x2e\x24\xc5\xe9\x73\x9a\x83\x23\x1f\xe3\xe1\xc9\x98\x9c\xf3\x3a\x3f\xac\x97\xa3\x1a\xcb\xcc\x26\x24\x25\xe4\x71\x1f\xbd\x23\xcb\x4c\xf0\xc3\xa8\x3e\xd6\x47\x1a\x87\xfc\xb2\x9c\x1c\x31\x18\x96\xcf\x97\xfa\xa6\x53\x84\x78\x5c\x3e\x06\x47\x77\x1b\x2e\x87\xaa\x70\x5f\xad\xc4\x12\xde\xc7\x69\xba\x02\x7d\x03\x00\xdd\xe1\x2d\x7a\xec\x1d\xbd\xa2\xdd\xae\x1e\x19\xbb\xcb\x94\x0d\x17\xa9\xac\x2f\x32\x7f\x68\xb1\x67\x50\xe4\x67\x15\xb2\xa8\xbf\x7f\xe8\x4d\x91\xdd\xa5\xc7\xbc\xcc\x43\x2d\x71\x63\x5d\x3c\x98\xf1\x5c\xe2\xdb\x2d\x18\xb2\x12\x48\xfb\x9f\x5a\xd1\x6b\x9c\xd1\xbe\xa9\x23\x24\xed\x9b\xef\x00\x00\x00\xff\xff\x0a\xc1\xa7\x81\x72\x01\x00\x00")

func assetsPauseSvgBytes() ([]byte, error) {
	return bindataRead(
		_assetsPauseSvg,
		"assets/pause.svg",
	)
}

func assetsPauseSvg() (*asset, error) {
	bytes, err := assetsPauseSvgBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/pause.svg", size: 370, mode: os.FileMode(436), modTime: time.Unix(1490523630, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsPlaySvg = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x8d\xd1\x4a\xc3\x40\x10\x45\xdf\xf3\x15\x97\xf1\x35\x76\xef\xae\x89\x11\xc9\x06\xf4\xdd\x8f\x10\x1a\x67\x83\x31\x29\xdd\x25\x5b\xfc\x7a\x69\x59\x29\x94\x32\xcc\xc0\xbd\x1c\xce\xf4\x71\x53\x9c\x7e\xe6\x25\x7a\x09\x29\x1d\x5e\x8d\xc9\x39\xef\xf2\xd3\x6e\x3d\xaa\x71\x24\x4d\xdc\x54\x90\xa7\x7d\x0a\x5e\x1c\x29\x08\xe3\xa4\x21\x95\xb0\x4d\x63\x7e\x5f\x4f\x5e\x08\xc2\xf1\xb2\x32\x54\xbd\x22\xa6\xe3\xfa\x3d\x7a\x79\x68\x78\x1e\x29\xc5\x63\x51\x59\x27\xf8\x9a\xe6\xf9\x0a\x0c\x15\x00\xf4\x87\xcf\x14\xb0\xf7\xf2\x81\x8e\xf5\x33\x61\x1b\xd6\x96\x3c\x27\xdb\x10\xbf\x62\x0a\xa8\x37\xc6\x97\x7f\xe1\xb2\x2e\x63\xb1\xdd\x18\x5d\x7b\x51\xbd\xa1\x6b\xeb\xae\x05\x61\x6b\x0b\xdb\xdd\x6b\x0b\x7a\x7d\x67\x74\xa8\xca\x89\x9b\x0e\xd5\x5f\x00\x00\x00\xff\xff\x74\x48\x45\x6f\x3b\x01\x00\x00")

func assetsPlaySvgBytes() ([]byte, error) {
	return bindataRead(
		_assetsPlaySvg,
		"assets/play.svg",
	)
}

func assetsPlaySvg() (*asset, error) {
	bytes, err := assetsPlaySvgBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/play.svg", size: 315, mode: os.FileMode(436), modTime: time.Unix(1490506353, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsPrevSvg = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x64\xcc\xc1\x6a\x84\x30\x10\x06\xe0\xbb\x4f\x31\x4c\xaf\xd6\xfc\x89\x5a\x4a\x49\x3c\xf4\xde\x87\x28\x68\x93\x50\x77\x15\x13\x12\xd9\xa7\x5f\x94\xc0\x2e\x2c\xc3\x0c\xfc\xc3\xc7\xaf\x43\xb2\xb4\x5f\xe6\x6b\x30\xec\x62\x5c\xbf\x84\xc8\x39\x37\xb9\x6d\x96\xcd\x0a\x05\x40\x84\x64\x99\xb2\x1f\xa3\x33\xac\x00\x26\x37\x79\xeb\x62\x09\xc9\x4f\xf9\x7b\xd9\x0d\x83\x40\x0a\xe7\xf2\x50\x69\x4b\x21\x6e\xcb\xff\x64\xf8\xad\xc3\x31\x5c\x1e\xef\xa5\x4a\x2a\xa6\x3f\x3f\xcf\x0f\x30\x54\x44\x44\x7a\xfd\x8d\x8e\x46\xc3\x3f\x24\xfb\xbe\xfe\x00\x7d\xf6\xb5\x04\xce\x24\x3b\xd0\x8d\xc5\xab\xc4\x29\xdb\x22\xf1\x24\xb5\xb0\xc7\x09\xc9\x0e\xd5\x3d\x00\x00\xff\xff\x27\x39\x48\xbf\xee\x00\x00\x00")

func assetsPrevSvgBytes() ([]byte, error) {
	return bindataRead(
		_assetsPrevSvg,
		"assets/prev.svg",
	)
}

func assetsPrevSvg() (*asset, error) {
	bytes, err := assetsPrevSvgBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/prev.svg", size: 238, mode: os.FileMode(436), modTime: time.Unix(1490506308, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsRandomSvg = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x8f\xd1\x6a\xc3\x20\x18\x85\xef\xf3\x14\x87\x7f\xb7\x59\xfd\x75\x9a\x95\x11\x03\xdb\xfd\x1e\x62\xd0\x4c\xc3\xb2\xa4\x54\x89\x65\x4f\x3f\x6c\x2c\xdd\x42\x11\x85\xff\xf8\x9d\x73\xb4\x0d\x8b\xc3\xf9\x7b\x9c\x82\x25\x1f\xe3\xf1\x45\x88\x94\xd2\x2e\x3d\xed\xe6\x93\x13\x8a\x99\x45\x58\x1c\x21\x0d\x87\xe8\x2d\x29\x66\x82\xef\x07\xe7\x63\x19\x96\xa1\x4f\x6f\xf3\xd9\x12\x83\xa1\xf8\xb2\xa9\xab\x5a\x87\x10\x4f\xf3\x57\x6f\xe9\x41\x73\x5e\x54\x84\xc7\x12\x25\x15\xe1\x73\x18\xc7\x1b\xd0\x55\x00\xd0\x1e\x3f\xa2\xc7\xc1\xd2\x3b\xa4\x31\xb5\x32\x90\x7b\x53\x6b\xbe\x4c\xc6\xe0\x87\xc4\x5d\x50\xea\x95\x94\xcd\x8a\xca\xe7\xbf\xac\xdb\xb4\xef\xaf\xe5\xd3\x3c\xf5\xa5\x79\x13\xaa\x39\xb7\xbe\xa2\xe1\xba\xc9\x9f\x93\xb5\xcc\x5a\xce\xbf\xc6\x6e\x9f\xd1\xac\xd7\xff\x3d\x59\xd5\x37\x4f\x2b\x5c\x57\x95\x23\x2c\xae\xab\x7e\x03\x00\x00\xff\xff\xa2\x45\x96\xa5\x83\x01\x00\x00")

func assetsRandomSvgBytes() ([]byte, error) {
	return bindataRead(
		_assetsRandomSvg,
		"assets/random.svg",
	)
}

func assetsRandomSvg() (*asset, error) {
	bytes, err := assetsRandomSvgBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/random.svg", size: 387, mode: os.FileMode(436), modTime: time.Unix(1490535047, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsRepeatSvg = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x5c\x8f\xc1\x4e\xc3\x30\x10\x44\xef\xf9\x8a\xd1\x72\x2d\xf5\xc6\xad\xa5\x0a\xc5\x39\x70\xe7\x23\x10\x35\xeb\x88\x90\x54\xb1\x15\xa7\x7c\x3d\xb2\x71\x01\x55\x96\x3d\x1a\x6b\xf4\x76\xb6\x0b\xab\x60\xfb\x1c\xa7\x60\xc9\xc7\x78\x79\x52\x2a\xa5\xb4\x4f\x87\xfd\xbc\x88\xd2\xcc\xac\xc2\x2a\x84\x34\x9c\xa3\xb7\xa4\x99\x09\xde\x0d\xe2\x63\x35\xeb\xe0\xd2\xf3\xbc\x59\x62\x30\x34\x97\x4b\x7d\xd3\x09\x42\x5c\xe6\x0f\x67\xe9\xe1\xc8\xf9\x50\xfd\x78\xac\xa8\x56\x13\xde\x87\x71\xfc\x0b\xf4\x0d\x00\x74\x97\xd7\xe8\x71\xb6\xf4\x82\x93\xd9\x69\x83\xb6\x35\xbb\x23\x67\x63\x0c\xbe\x48\xd5\x98\xdc\xf1\x4e\x37\xdc\x34\x4f\xae\xb2\x4a\x70\x71\x6f\x11\x9b\xa5\x5c\xe1\xfa\x23\xbf\x15\xfe\x6d\x53\xcc\xb2\x59\x3a\x64\xbd\x16\xbd\xcd\x52\xd2\x37\xf5\x09\xab\xf4\xcd\x77\x00\x00\x00\xff\xff\x5e\x9e\x3f\xba\x36\x01\x00\x00")

func assetsRepeatSvgBytes() ([]byte, error) {
	return bindataRead(
		_assetsRepeatSvg,
		"assets/repeat.svg",
	)
}

func assetsRepeatSvg() (*asset, error) {
	bytes, err := assetsRepeatSvgBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/repeat.svg", size: 310, mode: os.FileMode(436), modTime: time.Unix(1490535180, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"assets/app.css": assetsAppCss,
	"assets/app.html": assetsAppHtml,
	"assets/app.js": assetsAppJs,
	"assets/next.svg": assetsNextSvg,
	"assets/pause.svg": assetsPauseSvg,
	"assets/play.svg": assetsPlaySvg,
	"assets/prev.svg": assetsPrevSvg,
	"assets/random.svg": assetsRandomSvg,
	"assets/repeat.svg": assetsRepeatSvg,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"assets": &bintree{nil, map[string]*bintree{
		"app.css": &bintree{assetsAppCss, map[string]*bintree{}},
		"app.html": &bintree{assetsAppHtml, map[string]*bintree{}},
		"app.js": &bintree{assetsAppJs, map[string]*bintree{}},
		"next.svg": &bintree{assetsNextSvg, map[string]*bintree{}},
		"pause.svg": &bintree{assetsPauseSvg, map[string]*bintree{}},
		"play.svg": &bintree{assetsPlaySvg, map[string]*bintree{}},
		"prev.svg": &bintree{assetsPrevSvg, map[string]*bintree{}},
		"random.svg": &bintree{assetsRandomSvg, map[string]*bintree{}},
		"repeat.svg": &bintree{assetsRepeatSvg, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
