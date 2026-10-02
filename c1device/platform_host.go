//go:build !linux || !mipsle

package c1device

// PC 模拟器后端：把 5624 字节的 1bpp 帧渲染到 SDL3 窗口，并还原墨水屏观感。
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
//	C1SIM_SCALE    窗口放大倍数，1..8，默认 4
//	C1SIM_TIMING   1（默认）模拟刷新时序：全刷走"白→黑→白"闪烁，约 700ms
//	               0 立即显示，便于截图与自动化
//	C1SIM_GHOST    残影灰度 0..255，越大越淡，255 等于关闭，默认 192
//	C1SIM_HEADLESS 1 无头模式：不开窗口，把帧导出成 PNG 后让应用正常退出
//	C1SIM_DUMP     无头模式的输出路径，默认 frame.png
//	               C1SIM_FRAMES>1 时作为文件名前缀，产出 <前缀>_0001.png 等序列
//	C1SIM_FRAMES   无头模式导出的帧数，1（默认）只导首帧到 C1SIM_DUMP；
//	               2..64 导出带序号的帧序列，并自动注入合成按键驱动画面变化
//	C1SIM_FRAME_DELAY 相序列两帧的间隔毫秒数，默认 120，避免无节流狂写磁盘
//	C1SIM_TIMEOUT  无头模式的总超时，默认 10s，到点强制退出，绝不挂住
import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

const (
	simDefaultScale = 4
	simMinScale     = 1
	simMaxScale     = 8

	simDefaultGhost = 192
)

// 刷新时序取自 theBillLee/c1-slim 的实测：写入约 150ms，全刷约 700ms。
const (
	fullRefreshMs  = 700
	partialWriteMs = 150
)

// 全刷闪烁的阶段划分（占 fullRefreshMs 的比例）：白 → 黑 → 白 → 内容
const (
	flashWhite1End = 0.30
	flashBlackEnd  = 0.55
	flashWhite2End = 0.85
)

// simDefaultDump 是无头模式默认的输出文件名（相对当前工作目录）。
const simDefaultDump = "frame.png"

// 无头模式的导出参数默认值与上界。
const (
	simDefaultFrames = 1
	simMaxFrames     = 64

	simDefaultFrameDelay = 120 * time.Millisecond
	simMinFrameDelay     = 0
	simMaxFrameDelay     = 5 * time.Second

	simDefaultTimeout = 10 * time.Second
)

// simScriptKeys 是帧序列模式下自动注入的按键序列，用来驱动应用重绘。
//
// 每个键都必须改变渲染结果：Draw 对相同内容会去重，若某个键不产生新内容，
// 序列就会停在那个画面上白等到超时。脚本在 frames 用尽后循环，因此它是
// "周期"而非"覆盖全部交互"——应用是有限状态机时，绕一圈后会回到相同画面，
// 这属于预期（重复帧本身也是有效的回归基线）。
var simScriptKeys = []Key{
	KeyDown, KeyOK, KeyUp, KeyPause,
	KeyVolumeUp, KeyVolumeDown, KeyOK, KeyDown,
}

// simOptions 是模拟器的运行参数，全部来自环境变量，在 OpenPlatform 时读取一次。
type simOptions struct {
	scale      int32
	timing     bool
	ghost      uint8
	headless   bool
	dump       string
	frames     int
	frameDelay time.Duration
	timeout    time.Duration
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

	ghost := envInt("C1SIM_GHOST", simDefaultGhost)
	if ghost < 0 {
		ghost = 0
	}
	if ghost > 255 {
		ghost = 255
	}

	dump := os.Getenv("C1SIM_DUMP")
	if dump == "" {
		dump = simDefaultDump
	}

	frames := envInt("C1SIM_FRAMES", simDefaultFrames)
	if frames < 1 {
		frames = 1
	}
	if frames > simMaxFrames {
		frames = simMaxFrames
	}

	delay := time.Duration(envInt("C1SIM_FRAME_DELAY", int(simDefaultFrameDelay/time.Millisecond))) * time.Millisecond
	if delay < simMinFrameDelay {
		delay = simMinFrameDelay
	}
	if delay > simMaxFrameDelay {
		delay = simMaxFrameDelay
	}

	timeout := time.Duration(envInt("C1SIM_TIMEOUT", int(simDefaultTimeout/time.Second))) * time.Second
	if timeout <= 0 {
		timeout = simDefaultTimeout
	}

	return simOptions{
		scale:      scale,
		timing:     envInt("C1SIM_TIMING", 1) != 0,
		ghost:      uint8(ghost),
		headless:   envInt("C1SIM_HEADLESS", 0) != 0,
		dump:       dump,
		frames:     frames,
		frameDelay: delay,
		timeout:    timeout,
	}
}

// writeGrayPNG 把每像素一字节的灰度缓冲写成 PNG（0x00 黑、0xFF 白）。
func writeGrayPNG(path string, gray []uint8) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	img := &image.Gray{
		Pix:    gray,
		Stride: DisplayWidth,
		Rect:   image.Rect(0, 0, DisplayWidth, DisplayHeight),
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// seqFramePath 把 C1SIM_DUMP 当成前缀，生成带序号的帧文件名：
// out/demo.png + 第 3 帧 → out/demo_0003.png。序号固定 4 位，天然按序排列。
func seqFramePath(base string, n int) string {
	ext := filepath.Ext(base)
	if ext == "" {
		ext = ".png"
	}
	return fmt.Sprintf("%s_%04d%s", strings.TrimSuffix(base, ext), n, ext)
}

// runHeadless 无头模式：不初始化 SDL，把帧导出成 PNG 后收工。
// 返回后 pump 的 defer 会关闭事件通道，应用据此正常退出（而不是报错）。
//
// 四道刹车保证一定停得下来：帧数上限、总超时、帧间隔、单帧等待上限。
func (p *hostPlatform) runHeadless() {
	gray := make([]uint8, DisplayWidth*DisplayHeight)
	for i := range gray {
		gray[i] = 0xFF
	}

	total := p.opts.frames
	if total < 1 {
		total = 1
	}
	exported := 0

	// writeFrame 导出一帧并计数；返回 false 表示写盘失败，应立即收工（不重试）。
	writeFrame := func(frame Frame) bool {
		// 无头导出的是"干净"内容：不做闪烁动画，也不叠残影
		DecodeGray(frame, gray)
		path := p.opts.dump
		if total > 1 {
			path = seqFramePath(p.opts.dump, exported+1)
		}
		if err := writeGrayPNG(path, gray); err != nil {
			fmt.Fprintf(os.Stderr, "headless: 导出失败: %v\n", err)
			return false
		}
		exported++
		fmt.Printf("headless: wrote %s (%dx%d) %d/%d\n",
			path, DisplayWidth, DisplayHeight, exported, total)
		return true
	}

	timeout := p.opts.timeout
	if timeout <= 0 {
		timeout = simDefaultTimeout // 零值 simOptions 也要有兜底，绝不立刻超时
	}
	deadline := time.After(timeout)
	keySeq := 0

	for {
		select {
		case req := <-p.draw:
			if !writeFrame(req.frame) {
				return
			}
			if exported >= total {
				return // 帧数够了，正常收工
			}

			// 还想要更多帧：注入下一个按键，让应用重绘出新内容。
			key := simScriptKeys[keySeq%len(simScriptKeys)]
			keySeq++
			select {
			case p.out <- Event{Key: key}:
			case <-p.quit:
				return
			}

			if p.opts.frameDelay > 0 {
				timer := time.NewTimer(p.opts.frameDelay)
				select {
				case <-timer.C:
				case <-p.quit:
					timer.Stop()
					return
				}
			}
		case <-p.quit:
			return
		case <-deadline:
			fmt.Fprintf(os.Stderr, "headless: 超时收工，已导出 %d/%d 帧\n", exported, total)
			return
		}
	}
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

// paintContent 把帧画进 RGBA 缓冲，并按上一次内容叠加残影。
//
// 残影规则：上一次是黑、这一次变白的像素，留一层灰而不是纯白。
// 全刷会清掉残影（applyGhost=false），这正是应用"每 12 次做一次全刷防残影"的意义所在。
// 返回更新后的"上一次黑点掩码"，供下一次计算残影。
func paintContent(pix, gray []uint8, frame Frame, ghostFrom []bool, ghost uint8, applyGhost bool) []bool {
	DecodeGray(frame, gray)

	n := DisplayWidth * DisplayHeight
	if ghostFrom == nil {
		ghostFrom = make([]bool, n)
	}

	for i := 0; i < n; i++ {
		v := gray[i]
		if applyGhost && v == 0xFF && ghostFrom[i] {
			v = ghost
		}
		pix[i*4], pix[i*4+1], pix[i*4+2] = v, v, v
		ghostFrom[i] = gray[i] == 0x00
	}
	return ghostFrom
}

// paintFlat 把整屏填成同一灰度（全刷闪烁用）。
func paintFlat(pix []byte, v uint8) {
	for i := 0; i < len(pix)/4; i++ {
		pix[i*4], pix[i*4+1], pix[i*4+2] = v, v, v
	}
}

// pump 是唯一接触 SDL 的 goroutine。
func (p *hostPlatform) pump() {
	runtime.LockOSThread()

	defer close(p.done)
	// 关闭 out 之前已投递的事件仍可被应用读到（Go 带缓冲 channel 的语义），
	// 所以"关窗 = 按返回键"能让应用走正常退出路径而不是报错。
	defer close(p.out)

	// 无头模式在触碰 SDL 之前就分流：完全不初始化 SDL，不需要窗口也不需要 DLL。
	if p.opts.headless {
		p.runHeadless()
		return
	}

	defer sdl.Quit()
	if !sdl.Init(sdl.InitVideo) {
		return
	}

	scale := p.opts.scale

	var window *sdl.Window
	var renderer *sdl.Renderer
	if !sdl.CreateWindowAndRenderer(simWindowTitle(scale),
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

	// 当前按住的键 → 下一次产生重复事件的时间（仅记录可重复的键）
	held := map[sdl.Scancode]time.Time{}

	// 显示状态
	var (
		cur         Frame  // 待显示/正在显示的内容
		haveContent bool   // 是否已经收到过一帧
		ghostFrom   []bool // 上一次内容的黑点掩码，用于算残影
		flashing    bool   // 是否正在走全刷闪烁
		flashStart  time.Time
		postFlash   bool // 闪烁刚结束，本帧应以"无残影"的干净画面呈现
	)

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
						sdl.SetWindowTitle(window, simWindowTitle(scale)) // 标题里的倍数要跟着变
						dst.W = float32(DisplayWidth * scale)
						dst.H = float32(DisplayHeight * scale)
					}
					continue
				}
				if key.Repeat {
					continue // 重复由下面按软件节奏产生，忽略 OS 的重复事件
				}
				if ev, ok := mapSDLScancode(key.Scancode); ok {
					p.emit(ev)
					if repeatableKey(ev.Key) {
						if _, already := held[key.Scancode]; !already {
							held[key.Scancode] = time.Now().Add(repeatDelay)
						}
					}
				}
			case sdl.EventKeyUp:
				delete(held, event.Key().Scancode)
			case sdl.EventWindowFocusLost:
				// 失焦后收不到 keyup，必须清空，否则会一直重复下去
				for sc := range held {
					delete(held, sc)
				}
			}
		}

		// 1b) 软件自动重复：长按导航/音量键时持续产生 Repeat=true 的事件
		if len(held) > 0 {
			now := time.Now()
			for sc, next := range held {
				if now.Before(next) {
					continue
				}
				if ev, ok := mapSDLScancode(sc); ok && repeatableKey(ev.Key) {
					p.emit(Event{Key: ev.Key, Repeat: true})
				}
				held[sc] = now.Add(repeatInterval)
			}
		}

		// 2) 取最新的绘制请求（cap=1，最新胜）
		select {
		case req := <-p.draw:
			cur = req.frame
			haveContent = true
			if req.full {
				if p.opts.timing {
					flashing = true
					flashStart = time.Now()
				} else {
					postFlash = true // 不做动画，但语义上仍是"全刷后画面干净"
				}
			}
		default:
		}

		// 3) 计算本帧该显示什么
		switch {
		case flashing:
			elapsed := float64(time.Since(flashStart).Milliseconds()) / float64(fullRefreshMs)
			switch {
			case elapsed < flashWhite1End:
				paintFlat(pix, 0xFF)
			case elapsed < flashBlackEnd:
				paintFlat(pix, 0x00)
			case elapsed < flashWhite2End:
				paintFlat(pix, 0xFF)
			default:
				flashing = false
				postFlash = true
				ghostFrom = paintContent(pix, gray, cur, ghostFrom, p.opts.ghost, false)
			}
		case haveContent && postFlash:
			postFlash = false
			ghostFrom = paintContent(pix, gray, cur, ghostFrom, p.opts.ghost, false)
		case haveContent:
			ghostFrom = paintContent(pix, gray, cur, ghostFrom, p.opts.ghost, true)
		}

		// 4) 上屏
		if haveContent || flashing {
			sdl.UpdateTexture(texture, nil, unsafe.Pointer(&pix[0]), DisplayWidth*4)
		}
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

// simWindowTitle 窗口标题：始终显示当前放大倍数与常用按键提示，
// 免得忘掉 Ctrl+- 这类模拟器自身的快捷键（完整说明见 README.md）。
func simWindowTitle(scale int32) string {
	return fmt.Sprintf("C1-Slim Simulator %dx | Esc退出 | Ctrl+-缩小 Ctrl+=放大 Ctrl+0原尺寸 | 用法见 README.md", scale)
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
