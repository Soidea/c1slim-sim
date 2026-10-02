//go:build !linux || !mipsle

package c1device

// PC 模拟器后端：把 5624 字节的 1bpp 帧渲染到 SDL3 窗口。
//
// 设计要点
//
//  1. 与真机一致的语义：帧去重、首帧强制全刷（见 Draw）。
//  2. 线程模型：SDL 的建窗、渲染、事件泵必须在同一个 OS 线程。
//     因此 pump() 的第一条语句就是 runtime.LockOSThread()，且**所有** SDL 调用
//     都只发生在这个 goroutine 里；Draw() 只往 channel 投递帧，绝不碰 SDL。
//  3. 双端分派：本文件带 !linux || !mipsle，交叉编译到 linux/mipsle 时不参与编译，
//     因此真机产物不含任何 SDL 依赖（也不需要 cgo）。
//
// 模拟器开关（环境变量，详见 README）：
//
//	C1SIM_SCALE   窗口放大倍数，1..8，默认 4
import (
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

const (
	simDefaultScale = 4
	simMinScale     = 1
	simMaxScale     = 8
)

// simOptions 是模拟器的运行参数，全部来自环境变量，在 OpenPlatform 时读取一次。
type simOptions struct {
	scale int32
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func readSimOptions() simOptions {
	scale := int32(envInt("C1SIM_SCALE", simDefaultScale))
	if scale < simMinScale {
		scale = simMinScale
	}
	if scale > simMaxScale {
		scale = simMaxScale
	}
	return simOptions{scale: scale}
}

type drawReq struct {
	frame Frame
	full  bool
}

type hostPlatform struct {
	out  chan Event
	draw chan drawReq
	quit chan struct{}
	done chan struct{}

	opts simOptions

	last  Frame // 仅由 Draw（应用 goroutine）访问
	ready bool
	close sync.Once
}

// OpenPlatform 与真机实现同名，靠 build tag 做文件级分派。
func OpenPlatform() (Platform, error) {
	p := &hostPlatform{
		out:  make(chan Event, 32),
		draw: make(chan drawReq, 1),
		quit: make(chan struct{}),
		done: make(chan struct{}),
		opts: readSimOptions(),
	}
	go p.pump()
	return p, nil
}

// pump 是唯一接触 SDL 的 goroutine。
func (p *hostPlatform) pump() {
	runtime.LockOSThread()

	defer close(p.done)
	// 关闭 out 之前已投递的事件仍可被应用读到（Go 带缓冲 channel 的语义），
	// 所以"关窗 = 按返回键"能让应用走正常退出路径而不是报错。
	defer close(p.out)

	defer sdl.Quit()
	if !sdl.Init(sdl.InitVideo) {
		return
	}

	scale := p.opts.scale

	var window *sdl.Window
	var renderer *sdl.Renderer
	if !sdl.CreateWindowAndRenderer("C1-Slim Simulator",
		DisplayWidth*scale, DisplayHeight*scale, 0, &window, &renderer) {
		return
	}
	defer sdl.DestroyWindow(window)
	defer sdl.DestroyRenderer(renderer)

	texture := sdl.CreateTexture(renderer, sdl.PixelFormatRGBA8888,
		sdl.TextureAccessStreaming, DisplayWidth, DisplayHeight)
	if texture == nil {
		return
	}
	defer sdl.DestroyTexture(texture)
	// 最近邻：放大后保持 1bpp 的硬边，不被插值糊掉
	sdl.SetTextureScaleMode(texture, sdl.ScaleModeNearest)

	dst := sdl.FRect{
		W: float32(DisplayWidth * scale),
		H: float32(DisplayHeight * scale),
	}

	// RGBA 像素缓冲：alpha 恒定，灰度每次由帧解码写入
	pix := make([]byte, DisplayWidth*DisplayHeight*4)
	for i := 0; i < DisplayWidth*DisplayHeight; i++ {
		pix[i*4+3] = 0xFF
	}
	gray := make([]uint8, DisplayWidth*DisplayHeight)
	for i := range gray {
		gray[i] = 0xFF // 起始白屏
	}

	for {
		// 1) 抽干输入
		var event sdl.Event
		for sdl.PollEvent(&event) {
			switch event.Type() {
			case sdl.EventQuit:
				// 关窗口等价于按返回键
				p.emit(Event{Key: KeyBack})
				return
			case sdl.EventKeyDown:
				key := event.Key()
				// Ctrl 组合键是模拟器自身的控制键，不转发给应用
				if key.Mod&sdl.KeymodCtrl != 0 {
					if s, ok := simScaleKey(key.Scancode, scale); ok && s != scale {
						scale = s
						sdl.SetWindowSize(window, DisplayWidth*scale, DisplayHeight*scale)
						dst.W = float32(DisplayWidth * scale)
						dst.H = float32(DisplayHeight * scale)
					}
					continue
				}
				if key.Repeat {
					continue // 自动重复由模拟器自己按软件节奏产生，忽略 OS 重复
				}
				if ev, ok := mapSDLScancode(key.Scancode); ok {
					p.emit(ev)
				}
			case sdl.EventKeyUp:
				continue
			}
		}

		// 2) 取最新的绘制请求（cap=1，最新胜）
		select {
		case req := <-p.draw:
			DecodeGray(req.frame, gray)
			for i, v := range gray {
				pix[i*4], pix[i*4+1], pix[i*4+2] = v, v, v
			}
			sdl.UpdateTexture(texture, nil, unsafe.Pointer(&pix[0]), DisplayWidth*4)
			// req.full 目前只保留语义（真机用它触发墨水屏全刷动作）；
			// 全刷闪烁与残影的观感还原在下一步实现。
			_ = req.full
		default:
		}

		// 3) 上屏
		sdl.SetRenderDrawColor(renderer, 0, 0, 0, 0xFF)
		sdl.RenderClear(renderer)
		sdl.RenderTexture(renderer, texture, nil, &dst)
		sdl.RenderPresent(renderer)

		select {
		case <-p.quit:
			return
		default:
		}

		time.Sleep(8 * time.Millisecond) // 约 120fps，够跟手又不空转
	}
}

// simScaleKey 把 Ctrl 组合键映射成新的放大倍数。
//
//	Ctrl+= 放大   Ctrl+- 缩小   Ctrl+0 回到 1:1
func simScaleKey(sc sdl.Scancode, cur int32) (int32, bool) {
	switch sc {
	case sdl.ScancodeEquals:
		return min(cur+1, simMaxScale), true
	case sdl.ScancodeMinus:
		return max(cur-1, simMinScale), true
	case sdl.Scancode0:
		return 1, true
	}
	return cur, false
}

// emit 非阻塞投递：队列满时丢弃最旧的一条，避免 SDL 线程被卡住。
func (p *hostPlatform) emit(ev Event) {
	select {
	case p.out <- ev:
	default:
		select {
		case <-p.out:
		default:
		}
		select {
		case p.out <- ev:
		default:
		}
	}
}

// Draw 与真机保持同样语义：相同帧去重、首帧强制全刷。
func (p *hostPlatform) Draw(next Frame, full bool) error {
	if p.ready && next == p.last {
		return nil
	}
	full = full || !p.ready
	p.last, p.ready = next, true

	select {
	case <-p.draw:
	default:
	}
	select {
	case p.draw <- drawReq{frame: next, full: full}:
	case <-p.quit:
	}
	return nil
}

func (p *hostPlatform) Events() <-chan Event { return p.out }

func (p *hostPlatform) Close() error {
	p.close.Do(func() {
		close(p.quit)
		<-p.done
	})
	return nil
}
