package vv

import (
	"crypto/md5"
	_ "embed"
	"encoding/hex"
)

var (
	//go:embed index.html
	indexHTML     []byte
	indexHTMLHash = func() string {
		hasher := md5.New()
		hasher.Write(indexHTML)
		return hex.EncodeToString(hasher.Sum(nil))
	}
)
