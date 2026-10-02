// genframes 生成确定性的 5624 字节 golden 帧，供 frame2png.py（Python oracle）
// 与 Go 解码器交叉比对。每个用例专门捕捉一类帧格式错误。
package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"c1device"
)

const (
	W = c1device.DisplayWidth
	H = c1device.DisplayHeight
)

func set(f *c1device.Frame, x, y int) {
	f[(y/8)*W+x] |= 0x80 >> uint(y&7)
}

func write(out, name string, f *c1device.Frame) {
	if err := os.WriteFile(filepath.Join(out, name+".bin"), f[:], 0o644); err != nil {
		log.Fatal(err)
	}
}

func fillRows(f *c1device.Frame, y int) {
	for x := 0; x < W; x++ {
		set(f, x, y)
	}
}

func fillCols(f *c1device.Frame, x int) {
	for y := 0; y < H; y++ {
		set(f, x, y)
	}
}

func main() {
	out := flag.String("out", "../tests/golden", "golden 输出目录")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	// 全白 / 全黑 —— 抓位极性反了
	var allWhite c1device.Frame
	write(*out, "all_white", &allWhite)

	var allBlack c1device.Frame
	for i := range allBlack {
		allBlack[i] = 0xFF
	}
	write(*out, "all_black", &allBlack)

	// 行 0 / 7 / 8 —— 抓位序 MSB/LSB 反 + strip 边界
	for _, spec := range []struct {
		name string
		y    int
	}{{"row0", 0}, {"row7", 7}, {"row8", 8}, {"row151", H - 1}} {
		var f c1device.Frame
		fillRows(&f, spec.y)
		write(*out, spec.name, &f)
	}

	// 列 0 / 295 —— 抓行宽用错（误拿 152 当宽度）
	for _, spec := range []struct {
		name string
		x    int
	}{{"col0", 0}, {"col295", W - 1}} {
		var f c1device.Frame
		fillCols(&f, spec.x)
		write(*out, spec.name, &f)
	}

	// 四角 —— 综合边界
	var corners c1device.Frame
	set(&corners, 0, 0)
	set(&corners, W-1, 0)
	set(&corners, 0, H-1)
	set(&corners, W-1, H-1)
	write(*out, "corners", &corners)

	// 1×1 棋盘 —— 抓 offset 公式错、行列互换
	var checker1 c1device.Frame
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if (x+y)%2 == 0 {
				set(&checker1, x, y)
			}
		}
	}
	write(*out, "checker1", &checker1)

	// 8×8 方块棋盘 —— 抓 strip 错（y/8 写成 y%8）
	var checker8 c1device.Frame
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if (x/8+y/8)%2 == 0 {
				set(&checker8, x, y)
			}
		}
	}
	write(*out, "checker8", &checker8)

	// 递增字节 —— 8 个位位置全覆盖，抓 0x80>>y 写成 1<<y
	var bits c1device.Frame
	for i := range bits {
		bits[i] = byte(i)
	}
	write(*out, "bits", &bits)

	// 随机（固定种子）—— 整体压力
	var rnd c1device.Frame
	r := rand.New(rand.NewSource(1))
	for i := range rnd {
		rnd[i] = byte(r.Intn(256))
	}
	write(*out, "random", &rnd)
}
