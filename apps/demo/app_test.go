package main

import (
	"testing"

	"c1device"
)

func newTestApp(t *testing.T) *demoApp {
	t.Helper()
	face, err := loadFace(16)
	if err != nil {
		t.Fatalf("加载点阵字体失败: %v", err)
	}
	t.Cleanup(func() { face.Close() })
	return &demoApp{face: face, cursor: 0, selected: -1, message: "就绪"}
}

// 首屏不能是空白：标题栏、列表、外框都应有像素。
func TestRenderProducesContent(t *testing.T) {
	app := newTestApp(t)
	frame := app.render()

	black := 0
	for _, b := range frame {
		for bit := 0; bit < 8; bit++ {
			if b&(0x80>>uint(bit)) != 0 {
				black++
			}
		}
	}
	if black == 0 {
		t.Fatal("画面全白，标题/列表/外框都没画出来")
	}
	if black == c1device.DisplayWidth*c1device.DisplayHeight {
		t.Fatal("画面全黑，不正常")
	}
	t.Logf("黑色像素数: %d", black)
}

// 输入必须改变画面：按 ↓ 后光标行下移，两帧必须不同。
func TestDownMovesCursor(t *testing.T) {
	app := newTestApp(t)
	before := app.render()

	app.handleEvent(c1device.Event{Key: c1device.KeyDown})
	after := app.render()

	if before == after {
		t.Fatal("按 ↓ 后画面未变化：输入→状态→画面的链路不通")
	}
	if app.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", app.cursor)
	}
}

// 到边界不能再移动。
func TestCursorStopsAtBounds(t *testing.T) {
	app := newTestApp(t)
	for i := 0; i < 10; i++ {
		app.handleEvent(c1device.Event{Key: c1device.KeyDown})
	}
	if app.cursor != len(demoRows)-1 {
		t.Fatalf("下边界 cursor = %d, want %d", app.cursor, len(demoRows)-1)
	}
	for i := 0; i < 10; i++ {
		app.handleEvent(c1device.Event{Key: c1device.KeyUp})
	}
	if app.cursor != 0 {
		t.Fatalf("上边界 cursor = %d, want 0", app.cursor)
	}
}

// Back 必须让应用退出。
func TestBackExits(t *testing.T) {
	app := newTestApp(t)
	if !app.handleEvent(c1device.Event{Key: c1device.KeyBack}) {
		t.Fatal("Back 应返回 true（退出）")
	}
}

// OK 选中当前行。
func TestOKSelects(t *testing.T) {
	app := newTestApp(t)
	app.handleEvent(c1device.Event{Key: c1device.KeyDown})
	app.handleEvent(c1device.Event{Key: c1device.KeyOK})
	if app.selected != 1 {
		t.Fatalf("selected = %d, want 1", app.selected)
	}
}
