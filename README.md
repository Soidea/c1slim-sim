# C1-Slim / MP-D261 PC 模拟器

在 PC 上用 **SDL3** 窗口模拟快易典 C1-Slim（296×152 单色墨水屏）的显示效果。
**同一份 Go 应用源码**两端通用：

| 目标 | 命令 | 产物 |
|---|---|---|
| PC 模拟器 | `.\build.ps1 -Target sim` | Windows 窗口程序 |
| 无头单帧 | `.\build.ps1 -Target shot` | `build/shots/demo.png` |
| 无头帧序列 | `.\build.ps1 -Target seq` | `build/shots/seq/demo_0001.png` …（默认 16 帧） |
| 无头截图 | `.\build.ps1 -Target shot` | 不弹窗口，导出首帧 PNG → `build/shots/demo.png` |
| 真机 | `.\build.ps1 -Target device` | `linux/mipsle` 静态 ELF |

> ⚠️ **验证等级**：本项目已在 PC 模拟器与交叉编译层面验证，**尚未在实机上验收**
> （当前没有实体设备）。不要对外声称"已通过实机验收"。

---

## 快速开始

前置条件：

- **Go 1.26+**（本机装在 `D:\DevProgramsSDK\go1.26\bin`，不在 PATH 时 `build.ps1` 会自动使用）
- **SDL3 ≥ 3.2.0** 的 `SDL3.dll`（本机已有 `D:\DevProgramsSDK\SDL3\SDL3-3.4.16\x86_64-w64-mingw32\bin\SDL3.dll`）
- Python + Pillow + numpy（仅跑交叉校验时需要）

```powershell
cd c1slim-sim

.\build.ps1 -Target sim       # 构建模拟器（并自动把 SDL3.dll 复制到输出目录）
.\build\sim\demo.exe          # 运行 Demo

.\build.ps1 -Target device    # 交叉编译真机 ELF → build/device/demo
.\build.ps1 -Target check     # 单元测试 + device 隔离性 + 帧解码交叉校验
```

**两端都不需要 cgo**：SDL3 通过 purego 在运行时动态加载，不参与编译期链接。

---

## 使用方法（照着做一遍）

**1. 构建并运行**

```powershell
cd c1slim-sim
.\build.ps1 -Target sim
.\build\sim\demo.exe
```

**2. 看窗口标题拿提示**

标题会常驻显示当前放大倍数和快捷键，缩放时同步变化：

```
C1-Slim Simulator 4x | Esc退出 | Ctrl+-缩小 Ctrl+=放大 Ctrl+0原尺寸 | 用法见 README.md
```

**3. 试一遍交互**

| 想看什么 | 怎么做 |
|---|---|
| 全刷闪烁 | 刚启动时那次"白→黑→白"（约 700ms）就是 |
| 光标移动 | 按 ↑ / ↓ |
| 长按连续翻页 | **按住 ↓ 不放**，约 0.45 秒后开始连续下移 |
| 选中 | 按 Enter，当前行加外框 |
| 残影 | 来回按几次 ↓，注意反白轨迹上留下的浅灰 |
| 残影被擦净 | 数到第 12 次刷新会触发一次全刷闪屏，之后残影消失 |

**4. 窗口太大就调小**（不必重启，直接在窗口里按）

```
Ctrl + -   缩小一档（4→3→2→1）
Ctrl + 0   回到 1:1（296×152，真机实际尺寸）
Ctrl + =   放大
```

**5. 退出**：按 `Esc`，或直接关窗口（两者都走正常退出，不是崩溃）。

**6. 想固定用小窗口启动**

```powershell
$env:C1SIM_SCALE='2'; .\build\sim\demo.exe
```

---

## 操作键位

| 键盘 | 传给应用的事件 |
|---|---|
| ↑ ↓ ← → | `KeyUp` / `KeyDown` / `KeyLeft` / `KeyRight` |
| Enter / 小键盘 Enter | `KeyOK` |
| Esc | `KeyBack` |
| 空格 | `KeyPause` |
| Backspace | `KeyRune '\b'` |
| Tab | `KeyRune '\t'` |
| `-` `[` | `KeyVolumeDown` |
| `=` `]` | `KeyVolumeUp` |
| 字母 / 数字 | `KeyRune` 对应字符（小写） |
| 关闭窗口 | `KeyBack`（应用走正常退出路径，不是崩溃） |

**长按自动重复**：↑ ↓ ← → 与音量键支持长按连续触发（首次等待 450ms，之后每 90ms 一次），
复刻真机 evdev `value==2` 的行为。打字类按键（`KeyOK` / `KeyBack` / `KeyRune` 等）**不重复**，
以免干扰输入。窗口失焦时会自动清空按住状态，不会出现"卡住一直重复"。

---

## 模拟器开关

| 快捷键 | 作用 |
|---|---|
| `Ctrl` + `=` | 放大一档（上限 8） |
| `Ctrl` + `-` | 缩小一档（下限 1） |
| `Ctrl` + `0` | 回到 1:1 |

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `C1SIM_SCALE` | `4` | 窗口放大倍数，1..8 |
| `C1SIM_TIMING` | `1` | `1` 模拟刷新时序（全刷"白→黑→白"闪烁，约 700ms）；`0` 立即显示，便于截图与自动化 |
| `C1SIM_GHOST` | `192` | 残影灰度 0..255，越大越淡，`255` 等于关闭 |
| `C1SIM_HEADLESS` | `0` | `1` 无头模式：不开窗口，导出帧 PNG 后让应用正常退出 |
| `C1SIM_DUMP` | `frame.png` | 无头模式的输出路径（目录不存在会自动创建）；多帧时作为文件名前缀 |
| `C1SIM_FRAMES` | `1` | 无头模式导出帧数。`1` 只导首帧到 `C1SIM_DUMP`；`2..64` 导出带序号的帧序列并自动注入合成按键 |
| `C1SIM_FRAME_DELAY` | `120` | 帧序列两帧间隔（毫秒），避免无节流狂写磁盘 |
| `C1SIM_TIMEOUT` | `10` | 无头模式总超时（秒），到点强制退出 |

```powershell
$env:C1SIM_SCALE='2'; $env:C1SIM_TIMING='0'; .\build\sim\demo.exe
```

### 无头截图（不开窗口，导出单帧）

适合回归比对、CI、或只想把画面存成图片：

```powershell
.\build.ps1 -Target shot          # → build/shots/demo.png（296×152，1bpp 真实尺寸）

# 自定义输出路径
$env:C1SIM_HEADLESS='1'; $env:C1SIM_DUMP='D:\tmp\ui.png'; .\build\sim\demo.exe
```

要点：

- **完全不初始化 SDL**，因此**不需要窗口，也不需要 `SDL3.dll`**
- 导出的是"干净"内容：不走闪烁动画、不叠残影
- 导出后关闭事件通道，应用按正常退出路径结束（不是报错）；若 10 秒内没等到帧会超时退出，不会挂住

### 无头帧序列（不开窗口，导出多帧）

把一段交互过程导成一串带序号的 PNG，便于逐帧回归比对：

```powershell
.\build.ps1 -Target seq                       # → build/shots/seq/demo_0001..0016.png
$env:C1SIM_FRAMES='32'; .\build.ps1 -Target seq # 换帧数（上限 64）
```

`C1SIM_DUMP` 在多帧时当**前缀**用，序号固定 4 位零填充（`demo_0003.png`），天然按序排列。

**会不会一直截不停？** 不会，有四道刹车，任一触发就退出：帧数上限、总超时（`C1SIM_TIMEOUT`）、帧间隔、写盘失败立即终止。正常路径下几百毫秒到几秒内自己结束。

关于画面内容：

- 帧序列导出的同样是**干净帧**（无残影、无闪烁动画），可直接当回归基线
- 帧与帧的差异来自**自动注入的合成按键**（`simScriptKeys`：下/确认/上/Pause/音量+/音量-/确认/下）。
  没有按键就没有新帧——应用只在收到事件后才重绘，而 `Draw` 对相同内容会去重
- 因此按键脚本里**每个键都必须改变渲染结果**，否则序列会停在那个画面上白等到超时。
  当前脚本只含 Demo 一定会响应的键；**换成别的应用时要复核这个列表**
- 脚本用尽后循环。应用是有限状态机时，绕一圈会回到相同画面（第 16 帧里通常有 1～2
  帧与前面重复），这是预期行为——重复帧本身也是有效基线

### 墨水屏观感是怎么还原的

- **全刷闪烁**：收到 `full=true` 时走"白 → 黑 → 白 → 内容"，总时长约 700ms
  （时序取自 `theBillLee/c1-slim` 的实测：写入约 150ms、全刷约 700ms）
- **残影**：上一次是黑、这一次变白的像素留一层灰而不是纯白
- **全刷清除残影**：正因为如此，`apps/demo` 里"每 12 次做一次全刷"的惯例
  在模拟器上肉眼可见其意义——残影会周期性被擦干净

---

## 写一个新应用

以 `apps/demo` 为模板复制一份，只需关心 `render()` 与 `handleEvent()`：

```go
platform, _ := c1device.OpenPlatform()
defer c1device.ReturnToDesktop()
defer platform.Close()

platform.Draw(app.render(), true)        // 首帧全刷
refreshes := 0
for {
    select {
    case <-ctx.Done():
        return nil
    case ev, ok := <-platform.Events():
        if !ok { return nil }
        if app.handleEvent(ev) { return nil }   // 返回 true 即退出
        refreshes++
        platform.Draw(app.render(), refreshes%12 == 0)  // 每 12 次全刷一次，防残影
    }
}
```

画布能力（`c1device` 提供）：`NewCanvas()`、`NewBitmapFace()`、
`DrawText` / `DrawTextCentered` / `DrawTextRight` / `DrawInvertedTextBar`、
`DrawLine` / `DrawRect` / `InvertRect` / `FillRect` / `DrawScrollIndicator`、
`canvas.Frame(threshold)` 转成待显示的帧。

`apps/demo/assets/pkg-font.bin` 是真机同款点阵字体（含中文），直接 `//go:embed` 使用即可。

> ⚠️ 如果在 **Linux** 上运行模拟器，请确保**不要**设置环境变量 `C1_C1ANCHER_TERMINAL=1`——
> 真机上它用于退出时回到启动器（会 kill 父进程），在 PC 上这会杀掉你的 shell。

---

## 帧格式（三个独立来源交叉验证）

```
5624 字节 = 296 × (152/8)，strip-major 打包
offset = (y/8)*296 + x
位     = 0x80 >> (y&7)
bit=1 表示黑色，MSB 是该 strip 的最上面一行
```

三个来源互相独立且一致：

1. 上游 `text.go` 的 `Canvas.Frame()`：`output[(y/8)*DisplayWidth+x] |= 0x80 >> (y&7)`
2. `mason-yb-zhang/c1slim-toolkit` 的 `tools/frame2png.py`
3. 上游 `App/refresh-test`、`App/badapple` 的 `writePreview()`

---

## 没有实机，怎么证明画得对

`tools/oracle/frame2png.py` 是**第三方独立实现**的解码器（MIT）。我们对同一份
5624 字节帧分别渲染，再**逐像素**比对（不能比 PNG 字节——两侧压缩级别不同，字节必然不等）。

```powershell
.\build.ps1 -Target check
```

13 个 golden 用例各自专门捕捉一类格式错误：

| 用例 | 捕捉的 bug |
|---|---|
| `all_white` / `all_black` | 位极性反了 |
| `row0` / `row7` / `row8` | 位序 MSB/LSB 反 + strip 边界 |
| `row151` | 下边界（152 = 19×8 整除） |
| `col0` / `col295` | 行宽用错 |
| `corners` | 四角综合边界 |
| `checker1` / `checker8` | offset 公式错、行列互换 |
| `bits`（`f[i]=byte(i)`） | 8 个位位置全覆盖 |
| `random`（固定种子） | 随机压力 |

---

## 为什么 device 构建天然不含 SDL

host 侧文件都带 `//go:build !linux || !mipsle`，交叉编译 `linux/mipsle` 时**根本不参与编译**，
真机侧只有 `platform_device_linux_mipsle.go` 里的 `OpenPlatform()`。

可自行验证：

```powershell
$env:CGO_ENABLED='0'; $env:GOOS='linux'; $env:GOARCH='mipsle'; $env:GOMIPS='hardfloat'
go list -deps ./apps/demo | Select-String sdl    # 应无输出
```

器件产物形态：`ELF32 / 小端 / EXEC（无动态链接器）/ EM_MIPS`，符合启动器
"必须是真 MIPS ELF、拒绝脚本"的契约。

---

## 目录

```
c1device/        上游 App/c1device（原样复制）+ 模拟器后端改造
  platform_host.go   [重写] 引擎 + SDL3（tag: !linux || !mipsle）
  keymap_host.go     [新增] SDL 物理键 → Event
  decode.go          [新增] 1bpp 帧解码（两端共用）
apps/demo/       示例应用 / 新应用模板
tools/           golden 生成、Go 版 frame2png、Python oracle 与比对脚本
tests/golden/    确定性 golden 帧（*.bin）
third_party/     SDL3.dll（不入库，由 build.ps1 复制）
```

---

## 许可

本项目因改造上游 `App/c1device` 而沿用 **GPL-3.0**，详见 `LICENSE` 与 `NOTICE.md`。

- 字体 `pkg-font.bin`：SIL OFL 1.1（`font-LICENSE.txt` 必须一同分发）
- `tools/oracle/frame2png.py`：来自 c1slim-toolkit，MIT
- `purego-sdl3`：Unlicense（公有领域）

---

## 未实现（后续可加）

- 导出多帧（目前只导出首帧；可做序列编号用于动画/回归）
- 命令行参数形式（目前只用环境变量）
- 鼠标/触摸事件（C1 Max 是触屏，C1-Slim 只有键盘，故暂不需要）
