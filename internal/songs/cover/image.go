package cover

import (
	"bytes"
	"image"
	"time"

	_ "image/gif" // support gif cover load
	"image/jpeg"
	_ "image/png" // support png cover load
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"

	_ "golang.org/x/image/bmp" // support bmp cover load

	"golang.org/x/image/draw"
)

func resizeImage(data io.ReadSeeker, width, height int) ([]byte, error) {
	info, _, err := image.DecodeConfig(data)
	if err != nil {
		return nil, err
	}
	if _, err := data.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(data)
	if err != nil {
		return nil, err
	}
	imgRatio := float64(info.Width) / float64(info.Height)
	outRatio := float64(width) / float64(height)
	if imgRatio > outRatio {
		height = int(math.Round(float64(height*info.Height) / float64(info.Width)))
	} else {
		width = int(math.Round(float64(width*info.Width) / float64(info.Height)))
	}
	rect := image.Rect(0, 0, width, height)
	out := image.NewRGBA(rect)
	draw.CatmullRom.Scale(out, rect, img, img.Bounds(), draw.Over, nil)
	outwriter := new(bytes.Buffer)
	opt := jpeg.Options{Quality: 100}
	jpeg.Encode(outwriter, out, &opt)
	return outwriter.Bytes(), nil
}

func serveImage(rpath string, w http.ResponseWriter, r *http.Request) {
	i, err := os.Stat(rpath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !modifiedSince(r, i.ModTime()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	f, err := os.Open(rpath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	q := r.URL.Query()
	ws, hs := q.Get("width"), q.Get("height")
	if len(ws) == 0 || len(hs) == 0 {
		w.Header().Add("Cache-Control", "max-age=86400")
		w.Header().Add("Content-Length", strconv.FormatInt(i.Size(), 10))
		w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(rpath)))
		w.Header().Add("Last-Modified", i.ModTime().Format(http.TimeFormat))
		io.CopyN(w, f, i.Size())
		return
	}
	wi, err := strconv.Atoi(ws)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hi, err := strconv.Atoi(hs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	b, err := resizeImage(f, wi, hi)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Add("Cache-Control", "max-age=86400")
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(rpath)))
	w.Header().Add("Last-Modified", i.ModTime().Format(http.TimeFormat))
	w.Write(b)
}

// ext guesses image extention by binary.
func ext(b []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(b))
	return format, err
}

/*modifiedSince compares If-Modified-Since header given time.Time.*/
func modifiedSince(r *http.Request, l time.Time) bool {
	t, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err != nil {
		return true
	}
	return !l.Before(t.Add(time.Second))
}
