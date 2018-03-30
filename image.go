package main

import (
	"bytes"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"math"
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
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	info, _, err := image.DecodeConfig(bytes.NewReader(data))
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
