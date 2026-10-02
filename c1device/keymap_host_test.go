//go:build !linux || !mipsle

package c1device

import "testing"

// 只有导航与音量键可重复；打字类按键重复会干扰输入。
func TestRepeatableKey(t *testing.T) {
	repeatable := []Key{KeyUp, KeyDown, KeyLeft, KeyRight, KeyVolumeUp, KeyVolumeDown}
	for _, k := range repeatable {
		if !repeatableKey(k) {
			t.Fatalf("Key %v 应支持长按重复", k)
		}
	}
	notRepeatable := []Key{KeyOK, KeyBack, KeyPause, KeyRune}
	for _, k := range notRepeatable {
		if repeatableKey(k) {
			t.Fatalf("Key %v 不应自动重复（会误触/干扰输入）", k)
		}
	}
}
