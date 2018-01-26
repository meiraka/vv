package main

import (
	"bytes"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/bmp"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
)

func expandImage(data []byte, width, height int) ([]byte, error) {
	r := bytes.NewReader(data)
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	outRect := image.Rectangle{image.ZP, image.Pt(width, height)}
	out := image.NewRGBA(outRect)
	w := color.RGBA{255, 255, 255, 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			out.Set(x, y, w)
		}
	}
	s := img.Bounds().Size()
	l := image.Pt((width-s.X)/2, (height-s.Y)/2)
	target := image.Rectangle{l, image.Pt(l.X+s.X, l.Y+s.Y)}
	draw.Draw(out, target, img, image.ZP, draw.Over)
	outwriter := new(bytes.Buffer)
	png.Encode(outwriter, out)
	return outwriter.Bytes(), nil
}

func resizeImage(data []byte, width, height int) ([]byte, error) {
	r := bytes.NewReader(data)
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	out := resize.Thumbnail(uint(width), uint(height), img, resize.Bicubic)
	outwriter := new(bytes.Buffer)
	opt := jpeg.Options{Quality: 100}
	jpeg.Encode(outwriter, out, &opt)
	return outwriter.Bytes(), nil
}
