package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
)

func (h *HTTPHandlerConfig) assetsHandler(rpath string, b []byte, hash []byte) http.HandlerFunc {
	if h.LocalAssets {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Cache-Control", "max-age=1")
			http.ServeFile(w, r, rpath)
		}
	}
	m := mime.TypeByExtension(path.Ext(rpath))
	var gz []byte
	var err error
	if m != "image/png" && m != "image/jpg" {
		if gz, err = makeGZip(b); err != nil {
			log.Fatalf("failed to make gzip for static %s: %v", rpath, err)
		}
	}
	length := strconv.Itoa(len(b))
	gzLength := strconv.Itoa(len(gz))
	etag := fmt.Sprintf(`"%s"`, hash)
	lastModified := time.Now().Format(http.TimeFormat)
	return func(w http.ResponseWriter, r *http.Request) {
		if noneMatch(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Add("Cache-Control", "max-age=86400")
		if m != "" {
			w.Header().Add("Content-Type", m)
		}
		w.Header().Add("ETag", etag)
		w.Header().Add("Last-Modified", lastModified)
		if gz != nil {
			w.Header().Add("Vary", "Accept-Encoding")
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
				w.Header().Add("Content-Encoding", "gzip")
				w.Header().Add("Content-Length", gzLength)
				w.WriteHeader(http.StatusOK)
				w.Write(gz)
				return
			}
		}
		w.Header().Add("Content-Length", length)
		w.Write(b)
	}
}

func determineLanguage(r *http.Request, m language.Matcher) language.Tag {
	t, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	tag, _, _ := m.Match(t...)
	return tag
}

func (h *HTTPHandlerConfig) i18nAssetsHandler(rpath string, b []byte, hash []byte) http.HandlerFunc {
	matcher := language.NewMatcher(translatePrio)
	m := mime.TypeByExtension(path.Ext(rpath))
	if h.LocalAssets {
		return func(w http.ResponseWriter, r *http.Request) {
			info, err := os.Stat(rpath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			l := info.ModTime()
			if !modifiedSince(r, l) {
				w.WriteHeader(304)
				return
			}
			tag := determineLanguage(r, matcher)
			data, err := ioutil.ReadFile(rpath)
			if err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			data, err = translate(data, tag)
			if err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			w.Header().Add("Cache-Control", "max-age=1")
			w.Header().Add("Content-Language", tag.String())
			w.Header().Add("Content-Length", strconv.Itoa(len(data)))
			w.Header().Add("Content-Type", m+"; charset=utf-8")
			w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
			w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
			w.Write(data)
			return
		}
	}
	gz, err := makeGZip(b)
	if err != nil {
		log.Fatalf("failed to make gzip for static %s: %v", rpath, err)
	}
	bt := make([][]byte, len(translatePrio))
	gt := make([][]byte, len(translatePrio))
	for i := range translatePrio {
		data, err := translate(b, translatePrio[i])
		if err != nil {
			log.Fatalf("failed to translate %s to %v: %v", rpath, translatePrio[i], err)
		}
		bt[i] = data
		data, err = makeGZip(data)
		if err != nil {
			log.Fatalf("failed to translate %s to %v: %v", rpath, translatePrio[i], err)
		}
		gt[i] = data
	}
	etag := fmt.Sprintf(`"%s"`, hash)
	lastModified := time.Now().Format(http.TimeFormat)
	return func(w http.ResponseWriter, r *http.Request) {
		if noneMatch(r, etag) {
			w.WriteHeader(304)
			return
		}
		tag := determineLanguage(r, matcher)
		index := 0
		for ; index < len(translatePrio); index++ {
			if translatePrio[index] == tag {
				break
			}
		}
		b = bt[index]
		gz = gt[index]

		w.Header().Add("Cache-Control", "max-age=86400")
		w.Header().Add("Content-Language", tag.String())
		w.Header().Add("Content-Type", m+"; charset=utf-8")
		w.Header().Add("Etag", etag)
		w.Header().Add("Last-Modified", lastModified)
		w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
			w.Header().Add("Content-Encoding", "gzip")
			w.Header().Add("Content-Length", strconv.Itoa(len(gz)))
			w.Write(gz)
			return
		}
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.Write(b)
	}

}
