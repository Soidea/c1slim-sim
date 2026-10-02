// frame2png 用本项目的 Go 解码器把 5624 字节帧渲染成 1× PNG，
// 供 tools/oracle/pngdiff.py 与 Python oracle 的输出逐像素比对。
package main

import (
	"flag"
	"image"
	"image/png"
	"log"
	"os"

	"c1device"
)

func main() {
	in := flag.String("in", "", "输入 5624 字节帧文件")
	out := flag.String("out", "", "输出 PNG 路径")
	flag.Parse()

	if *in == "" || *out == "" {
		log.Fatal("usage: frame2png -in <frame.bin> -out <image.png>")
	}

	data, err := os.ReadFile(*in)
	if err != nil {
		log.Fatal(err)
	}
	if len(data) != c1device.FrameBytes {
		log.Fatalf("帧长度 %d，应为 %d", len(data), c1device.FrameBytes)
	}

	var frame c1device.Frame
	copy(frame[:], data)

	gray := c1device.DecodeGray(frame, nil)
	img := &image.Gray{
		Pix:    gray,
		Stride: c1device.DisplayWidth,
		Rect:   image.Rect(0, 0, c1device.DisplayWidth, c1device.DisplayHeight),
	}

	if err := os.MkdirAll(filepathDir(*out), 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func filepathDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "."
}
