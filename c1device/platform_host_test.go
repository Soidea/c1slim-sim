//go:build !linux || !mipsle

package c1device

import (
	"testing"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

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
