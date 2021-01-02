package vv

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/meiraka/vv/internal/gzip"
	"github.com/meiraka/vv/internal/http/request"
	"golang.org/x/text/language"
)

func i18nHandler(rpath string, b []byte, extra map[string]string) (http.HandlerFunc, error) {
	matcher := language.NewMatcher(translatePrio)
	m := mime.TypeByExtension(path.Ext(rpath))
	bt := make([][]byte, len(translatePrio))
	gt := make([][]byte, len(translatePrio))
	for i := range translatePrio {
		data, err := translate(b, translatePrio[i], extra)
		if err != nil {
			return nil, fmt.Errorf("translate %s %v: %w", rpath, translatePrio[i], err)
		}
		bt[i] = data
		data, err = gzip.Encode(data)
		if err != nil {
			return nil, fmt.Errorf("gzip %s %v: %v", rpath, translatePrio[i], err)
		}
		gt[i] = data
	}

	etag, err := etag(b, extra)
	if err != nil {
		return nil, fmt.Errorf("etag: %w", err)
	}
	lastModified := time.Now().Format(http.TimeFormat)
	return func(w http.ResponseWriter, r *http.Request) {
		if request.NoneMatch(r, etag) {
			w.WriteHeader(304)
			return
		}
		tag, index := determineLanguage(r, matcher)
		body := bt[index]
		gzBody := gt[index]

		w.Header().Add("Cache-Control", "max-age=86400")
		w.Header().Add("Content-Language", tag.String())
		w.Header().Add("Content-Type", m+"; charset=utf-8")
		w.Header().Add("Etag", etag)
		w.Header().Add("Last-Modified", lastModified)
		w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gzBody != nil {
			w.Header().Add("Content-Encoding", "gzip")
			w.Header().Add("Content-Length", strconv.Itoa(len(gzBody)))
			w.Write(gzBody)
			return
		}
		w.Header().Add("Content-Length", strconv.Itoa(len(body)))
		w.Write(body)
	}, nil
}

func i18nLocalHandler(rpath string, date time.Time, extra map[string]string) (http.HandlerFunc, error) {
	matcher := language.NewMatcher(translatePrio)
	m := mime.TypeByExtension(path.Ext(rpath))
	return func(w http.ResponseWriter, r *http.Request) {
		info, err := os.Stat(rpath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		l := info.ModTime().UTC()
		if l.Before(date) {
			l = date.UTC()
		}
		if !request.ModifiedSince(r, l) {
			w.WriteHeader(304)
			return
		}
		tag, _ := determineLanguage(r, matcher)
		data, err := ioutil.ReadFile(rpath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		data, err = translate(data, tag, extra)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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
	}, nil
}

func determineLanguage(r *http.Request, m language.Matcher) (language.Tag, int) {
	t, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	_, i, _ := m.Match(t...)
	return translatePrio[i], i
}

func etag(b []byte, extra map[string]string) (string, error) {
	// generate etag from b, translateData and extra
	hasher := md5.New()
	hasher.Write(b)
	jsonTD, err := json.Marshal(translateData)
	if err != nil {
		return "", fmt.Errorf("translateData: %w", err)
	}
	hasher.Write(jsonTD)
	jsonExtra, err := json.Marshal(extra)
	if err != nil {
		return "", fmt.Errorf("extra: %w", err)
	}
	hasher.Write(jsonExtra)
	return `"` + hex.EncodeToString(hasher.Sum(nil)) + `"`, nil
}

func translate(html []byte, lang language.Tag, extra map[string]string) ([]byte, error) {
	t, err := template.New("webpage").Parse(string(html))
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	m, ok := translateData[lang]
	if ok {
		for k, v := range extra {
			m[k] = v
		}
	} else {
		m = extra
	}
	err = t.Execute(buf, m)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
