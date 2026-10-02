// Demo：C1-Slim 应用的最小骨架，同时可作为新应用的开发模板。
// 同一份源码：go build 出 PC 模拟器，GOOS=linux GOARCH=mipsle 出真机 ELF。
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"c1device"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "demo:", err)
		os.Exit(1)
	}
}

func run() error {
	face, err := loadFace(16)
	if err != nil {
		return err
	}
	defer face.Close()

	app := &demoApp{face: face, cursor: 0, selected: -1, message: "就绪"}

	platform, err := c1device.OpenPlatform()
	if err != nil {
		return err
	}
	// ReturnToDesktop 在真机上用于回到启动器；PC 上为空实现。
	defer c1device.ReturnToDesktop()
	defer platform.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	if err := platform.Draw(app.render(), true); err != nil { // 首帧全刷
		return err
	}

	refreshes := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-platform.Events():
			if !ok {
				// 模拟器关窗时会先投递 Back 事件，这里只是兜底，按正常退出处理。
				return nil
			}
			if app.handleEvent(ev) {
				return nil
			}
			refreshes++
			// 每 12 次做一次全刷，防墨水屏残影（与上游 book-reader 同惯例）
			if err := platform.Draw(app.render(), refreshes%12 == 0); err != nil {
				return err
			}
		}
	}
}
