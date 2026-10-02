//go:build !linux || !mipsle

package c1device

import (
	"strings"
	"testing"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

// 窗口标题必须带上当前倍数（缩放后要同步更新）与快捷键提示。
func TestSimWindowTitle(t *testing.T) {
	title := simWindowTitle(3)
	if !strings.Contains(title, "3x") {
		t.Fatalf("标题应显示当前倍数, got %q", title)
	}
	if !strings.Contains(title, "README.md") {
		t.Fatalf("标题应提示查看 README.md, got %q", title)
	}
	if !strings.Contains(title, "Ctrl+-") {
		t.Fatalf("标题应提示缩小快捷键, got %q", title)
	}
	if simWindowTitle(3) == simWindowTitle(4) {
		t.Fatal("倍数变化时标题必须变化")
	}
}

func TestSimScaleKey(t *testing.T) {
	cases := []struct {
		name string
		from int32
		sc   sdl.Scancode
		want int32
	}{
		{"zoom in", 4, sdl.ScancodeEquals, 5},
		{"zoom out", 4, sdl.ScancodeMinus, 3},
		{"clamp at max", 8, sdl.ScancodeEquals, 8},
		{"clamp at min", 1, sdl.ScancodeMinus, 1},
		{"reset to 1:1", 6, sdl.Scancode0, 1},
	}
	for _, c := range cases {
		got, ok := simScaleKey(c.sc, c.from)
		if !ok {
			t.Fatalf("%s: not recognized as a scale key", c.name)
		}
		if got != c.want {
			t.Fatalf("%s: scale = %d, want %d", c.name, got, c.want)
		}
	}
}

// 缩放键必须只认 Ctrl 组合（由 pump 判定），普通方向键不能被吞掉。
func TestSimScaleKeyIgnoresOtherKeys(t *testing.T) {
	for _, sc := range []sdl.Scancode{sdl.ScancodeUp, sdl.ScancodeDown, sdl.ScancodeReturn, sdl.ScancodeEscape} {
		if _, ok := simScaleKey(sc, 4); ok {
			t.Fatalf("scancode %v 不应被当作缩放键（会吞掉应用输入）", sc)
		}
	}
}

// 残影：上一次是黑、这一次变白的像素应留一层灰，而不是纯白。
func TestPaintContentGhost(t *testing.T) {
	pix := make([]byte, DisplayWidth*DisplayHeight*4)
	gray := make([]uint8, DisplayWidth*DisplayHeight)

	var first Frame
	setPixel(&first, 0, 0, true)
	ghostFrom := paintContent(pix, gray, first, nil, 192, true)
	if pix[0] != 0x00 {
		t.Fatalf("新画的黑点应为 0x00, got %#x", pix[0])
	}

	var second Frame // 全白：刚才那个黑点现在变白
	ghostFrom = paintContent(pix, gray, second, ghostFrom, 192, true)
	if pix[0] != 192 {
		t.Fatalf("变白的旧黑点应留残影 192, got %#x", pix[0])
	}
	if ghostFrom[0] {
		t.Fatal("全白帧后 ghostFrom[0] 应为 false")
	}
}

// 全刷会清除残影——这正是应用"每 12 次全刷一次"的意义。
func TestFullRefreshClearsGhost(t *testing.T) {
	pix := make([]byte, DisplayWidth*DisplayHeight*4)
	gray := make([]uint8, DisplayWidth*DisplayHeight)

	var first Frame
	setPixel(&first, 0, 0, true)
	ghostFrom := paintContent(pix, gray, first, nil, 192, true)

	var second Frame
	ghostFrom = paintContent(pix, gray, second, ghostFrom, 192, false) // applyGhost=false
	if pix[0] != 0xFF {
		t.Fatalf("全刷后应无残影(纯白), got %#x", pix[0])
	}
	if ghostFrom[0] {
		t.Fatal("全刷后 ghostFrom 应反映当前帧（全白）")
	}
}

// ghost=255 等于关闭残影。
func TestGhostLevel255Disables(t *testing.T) {
	pix := make([]byte, DisplayWidth*DisplayHeight*4)
	gray := make([]uint8, DisplayWidth*DisplayHeight)

	var first Frame
	setPixel(&first, 5, 5, true)
	ghostFrom := paintContent(pix, gray, first, nil, 255, true)

	var second Frame
	idx := (5*DisplayWidth + 5) * 4
	paintContent(pix, gray, second, ghostFrom, 255, true)
	if pix[idx] != 0xFF {
		t.Fatalf("ghost=255 时不应有可见残影, got %#x", pix[idx])
	}
}

func TestPaintFlat(t *testing.T) {
	pix := make([]byte, DisplayWidth*DisplayHeight*4)
	paintFlat(pix, 0x00)
	for i := 0; i < DisplayWidth*DisplayHeight; i++ {
		if pix[i*4] != 0x00 {
			t.Fatalf("像素 %d 未填黑", i)
		}
	}
	paintFlat(pix, 0xFF)
	for i := 0; i < DisplayWidth*DisplayHeight; i++ {
		if pix[i*4] != 0xFF {
			t.Fatalf("像素 %d 未填白", i)
		}
	}
}

func TestReadSimOptionsTimingAndGhost(t *testing.T) {
	t.Setenv("C1SIM_TIMING", "0")
	if readSimOptions().timing {
		t.Fatal("C1SIM_TIMING=0 应关闭时序模拟")
	}
	t.Setenv("C1SIM_TIMING", "1")
	if !readSimOptions().timing {
		t.Fatal("C1SIM_TIMING=1 应开启时序模拟")
	}
	t.Setenv("C1SIM_GHOST", "999")
	if got := readSimOptions().ghost; got != 255 {
		t.Fatalf("ghost 应钳到 255, got %d", got)
	}
}

func TestReadSimOptionsClampsScale(t *testing.T) {
	t.Setenv("C1SIM_SCALE", "99")
	if got := readSimOptions().scale; got != simMaxScale {
		t.Fatalf("C1SIM_SCALE=99 should clamp to %d, got %d", simMaxScale, got)
	}
	t.Setenv("C1SIM_SCALE", "0")
	if got := readSimOptions().scale; got != simMinScale {
		t.Fatalf("C1SIM_SCALE=0 should clamp to %d, got %d", simMinScale, got)
	}
	t.Setenv("C1SIM_SCALE", "abc") // non-numeric falls back to default
	if got := readSimOptions().scale; got != simDefaultScale {
		t.Fatalf("invalid value should fall back to %d, got %d", simDefaultScale, got)
	}
}
