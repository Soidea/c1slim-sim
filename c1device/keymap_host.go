//go:build !linux || !mipsle

package c1device

// 模拟器的键盘映射：SDL 物理键（scancode）→ c1device.Event。
//
// 这里**不复用**包内的 mapKey()：它是面向真机 evdev 硬件键矩阵的私有函数，
// 且覆盖不全（没有 Esc / Space / Tab，缺大量字母）。模拟器直接构造 Event。
import (
	"time"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

const (
	// 长按后首次重复的等待时间，以及之后的重复间隔。
	// 取值参照常见实体键盘手感，让"长按翻页"和真机接近。
	repeatDelay    = 450 * time.Millisecond
	repeatInterval = 90 * time.Millisecond
)

// repeatableKey 判定哪些键支持长按自动重复。
// 真机由 evdev 的 value==2 提供；这里用软件节奏复刻。
// 只有导航与音量键重复——打字类按键重复会干扰输入。
func repeatableKey(k Key) bool {
	switch k {
	case KeyUp, KeyDown, KeyLeft, KeyRight, KeyVolumeUp, KeyVolumeDown:
		return true
	}
	return false
}

func mapSDLScancode(sc sdl.Scancode) (Event, bool) {
	switch sc {
	// 导航 / 操作
	case sdl.ScancodeUp:
		return Event{Key: KeyUp}, true
	case sdl.ScancodeDown:
		return Event{Key: KeyDown}, true
	case sdl.ScancodeLeft:
		return Event{Key: KeyLeft}, true
	case sdl.ScancodeRight:
		return Event{Key: KeyRight}, true
	case sdl.ScancodeReturn, sdl.ScancodeKpEnter:
		return Event{Key: KeyOK}, true
	case sdl.ScancodeEscape:
		return Event{Key: KeyBack}, true
	case sdl.ScancodeSpace:
		return Event{Key: KeyPause}, true

	// 音量（真机为机身侧键）
	case sdl.ScancodeMinus, sdl.ScancodeLeftBracket:
		return Event{Key: KeyVolumeDown}, true
	case sdl.ScancodeEquals, sdl.ScancodeRightBracket:
		return Event{Key: KeyVolumeUp}, true

	// 编辑键
	case sdl.ScancodeBackspace:
		return Event{Key: KeyRune, Rune: '\b'}, true
	case sdl.ScancodeTab:
		return Event{Key: KeyRune, Rune: '\t'}, true
	}

	if r, ok := scancodeRune(sc); ok {
		return Event{Key: KeyRune, Rune: r}, true
	}
	return Event{}, false
}

// scancodeRune 把字母与数字行映射成 rune。用物理键而非字符事件，
// 保证不同键盘布局下行为一致。
func scancodeRune(sc sdl.Scancode) (rune, bool) {
	switch {
	case sc >= sdl.ScancodeA && sc <= sdl.ScancodeZ:
		return rune('a' + int(sc-sdl.ScancodeA)), true
	case sc >= sdl.Scancode1 && sc <= sdl.Scancode9:
		return rune('1' + int(sc-sdl.Scancode1)), true
	case sc == sdl.Scancode0:
		return '0', true
	}
	return 0, false
}
