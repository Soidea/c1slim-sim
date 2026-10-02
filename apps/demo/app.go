package main

import (
	_ "embed"
	"fmt"
	"image"

	"c1device"
)

// pkg-font.bin 取自上游 App/book-reader/assets：C1BF 点阵，含中文，SIL OFL 1.1
// （许可文件 font-LICENSE.txt 与之同目录，必须一起分发）。
//
//go:embed assets/pkg-font.bin
var fontData []byte

func loadFace(lineHeight int) (*c1device.Face, error) {
	return c1device.NewBitmapFace(fontData, lineHeight)
}

var demoRows = []string{"1. 屏幕自检", "2. 残影测试", "3. 关于"}

type demoApp struct {
	face     *c1device.Face
	cursor   int // 当前高亮行
	selected int // -1 表示未选
	message  string
}

// render 画一屏。这是新应用的开发模板：改这里就是改界面。
func (a *demoApp) render() c1device.Frame {
	c := c1device.NewCanvas() // 初始全白

	// 标题栏（反白）
	c.DrawInvertedTextBar(a.face, image.Rect(0, 0, c1device.DisplayWidth, 20), "C1-SLIM SIM DEMO")

	// 列表：光标行反白，已选中行加外框
	for i, row := range demoRows {
		r := image.Rect(4, 26+i*26, c1device.DisplayWidth-4, 26+i*26+24)
		c.DrawText(a.face, r.Min.X+4, r.Min.Y+6, row)
		if i == a.cursor {
			c.InvertRect(r)
		}
		if i == a.selected {
			c.DrawRect(r.Inset(-1))
		}
	}

	// 分隔线 + 操作提示 + 状态行
	c.DrawLine(4, 112, c1device.DisplayWidth-5, 112)
	c.DrawText(a.face, 4, 118, "上下选择 OK确认 ESC退出")
	c.DrawText(a.face, 4, 134, a.message)

	c.DrawRect(image.Rect(0, 0, c1device.DisplayWidth, c1device.DisplayHeight)) // 外框
	return c.Frame(128)                                                         // 阈值与真机一致
}

// handleEvent 返回 true 表示退出应用。
func (a *demoApp) handleEvent(ev c1device.Event) bool {
	switch ev.Key {
	case c1device.KeyUp:
		if a.cursor > 0 {
			a.cursor--
		}
		a.message = "向上"
	case c1device.KeyDown:
		if a.cursor < len(demoRows)-1 {
			a.cursor++
		}
		a.message = "向下"
	case c1device.KeyOK:
		a.selected = a.cursor
		a.message = "选中 " + demoRows[a.cursor]
	case c1device.KeyBack:
		return true
	case c1device.KeyPause:
		a.message = "PAUSE"
	case c1device.KeyVolumeUp:
		a.message = "音量+"
	case c1device.KeyVolumeDown:
		a.message = "音量-"
	case c1device.KeyRune:
		a.message = fmt.Sprintf("按键 %q", ev.Rune)
	}
	return false
}
