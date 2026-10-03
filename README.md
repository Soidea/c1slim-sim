# C1-Slim / MP-D261 PC 模拟器

> **仓库名含义**：`c1slim-sim` = **C1-Slim**（`c1slim`，取自设备/固件名 `c1-slim`）+ **sim**（simulator 缩写），即「C1-Slim 模拟器」。本仓是模拟器的**自有仓**，请勿与上游带连字符的 `c1-slim`（C 固件仓）混淆——那是别人的官方仓库，只作 fetch 上游用。

在 PC 上用 **SDL3** 窗口模拟快易典 C1-Slim（296×152 单色墨水屏）的显示效果。
**同一份 Go 应用源码**两端通用：

| 目标 | 命令 | 产物 |
|---|---|---|
| PC 模拟器 | `.\build.ps1 -Target sim` | Windows 窗口程序 |
| 无头单帧 | `.\build.ps1 -Target shot` | `build/shots/demo.png` |
| 无头帧序列 | `.\build.ps1 -Target seq` | `build/shots/seq/demo_0001.png` …（默认 16 帧） |
| 双轮回归 | `.\build.ps1 -Target regress` | 导两轮逐帧比对，不一致则退出码 1 |
| 无窗口构建 | `.\build.ps1 -Target headless` | 零 SDL 依赖的 headless 二进制（CI 用）→ `build/headless/demo.exe` |
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

# 新机器（或重新克隆后）先跑一次初始化：启用 pre-commit 钩子 + 体检依赖
.\setup.ps1

.\build.ps1 -Target sim       # 构建模拟器（并自动把 SDL3.dll 复制到输出目录）
.\build\sim\demo.exe          # 运行 Demo

.\build.ps1 -Target device    # 交叉编译真机 ELF → build/device/demo
.\build.ps1 -Target check     # 单元测试 + device 隔离性 + 帧解码交叉校验
```

> `setup.ps1` 幂等，可重复运行。它做的两件事**不会随 `git clone` 自动生效**，需每台机器做一次：
> 启用仓库钩子（`core.hooksPath`，存在 `.git/config` 里），
> 以及体检 Go / SDL3 / Python 前置依赖。详见「提交前检查」一节。

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

> **模拟器与真机的字符集不同**：模拟器有完整键盘（a–z、0–9、Tab 等），
> 真机只有 `qwertyuio` + `z` `v` `.` + 数字。这是硬件差异，不是 bug。
> **控制键（↑↓←→ / Enter / Esc / 空格 / Backspace / 音量）两端必须完全一致**，
> 由 `TestHostAndDeviceAgreeOnControlKeys` 交叉断言——双端通用的核心承诺是
> "同一物理按键产生同一 Event"，漏一个就会变成"模拟器正常、真机失效"。

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

### 无窗口构建（`-tags headless`，CI 友好）

前面所有无头用法都是**运行时**分流：程序仍按默认方式链接 SDL3，只是靠
`C1SIM_HEADLESS=1` 在碰 SDL 之前就改走导出路径。所以 `-Target shot/seq/regress`
**运行时本来就不需要 SDL3**（无头分支从不调用 `sdl.Init`）。

`-tags headless` 则是**编译期**变体：整条 host 代码路径换成不链接 SDL3 的实现，
产出的二进制**零 SDL 依赖**（不需要 `SDL3.dll`/`.so`，也不涉及 cgo），可放进最小 CI
镜像或受限沙箱里跑。两种 host 变体由互斥的 build tag 选择：

| 构建 | 编译的 host 文件 | 需要 SDL3 | 用途 |
|---|---|---|---|
| 默认 | `platform_host.go` + `platform_window.go` + `keymap_window.go` | 是（运行时 purego 动态加载） | 交互式窗口 |
| `-tags headless` | `platform_host.go` + `platform_headless.go` + `keymap_host.go` | **否** | CI / 渲染回归 / 无显卡 |
| `linux/mipsle` | 不编译任何 host 文件 | 否 | 真机（路径不变） |

```powershell
.\build.ps1 -Target headless        # 构建零 SDL 依赖的 headless 二进制（自带一次冒烟导出）

# 该变体整个程序都是无头的，不需要 C1SIM_HEADLESS；DUMP/FRAMES 照常生效
$env:C1SIM_DUMP='D:\tmp\ui.png'; $env:C1SIM_FRAMES='4'
.\build\headless\demo.exe           # 无需任何 SDL3 即可导出 4 帧
```

也可以直接用 Go（交叉引用 `../../c1device`）：

```powershell
cd apps/demo
go build -tags headless -o ../../build/headless/demo-headless.exe .
```

要点：

- 该构建下**整个程序都是无头的**，不需要（也不看）`C1SIM_HEADLESS`；
  `C1SIM_DUMP` / `C1SIM_FRAMES` / `C1SIM_FRAME_DELAY` / `C1SIM_TIMEOUT` 照常生效。
- 可审计地确认"确实没链 SDL"：
  ```powershell
  cd c1device; go list -tags headless -deps . | Select-String purego-sdl3   # 应无输出
  ```
- 想给 CI 一个"绝不含 SDL"的产物时用 `-tags headless`；只是想在本机导几张图，
  直接 `-Target shot` 即可，不必加 tag。

### 双轮逐帧回归（确定性检查）

把帧序列导两遍，逐帧比对像素，验证渲染是**确定性的**（没有随机状态、时间戳、
未初始化内存等）。任何一帧不一致都会打印差异像素数并以**退出码 1** 结束，可直接用作 CI 闸门：

```powershell
.\build.ps1 -Target regress                    # 16 帧 × 2 轮
$env:C1SIM_FRAMES='32'; .\build.ps1 -Target regress
```

产物在 `build/shots/regress/round1|round2/`，可以直接肉眼 diff。

会检查两类问题：

| 报告 | 含义 |
|---|---|
| `DIFF` | 同一帧两轮像素不同 → 渲染不确定 |
| `SIZE` | 尺寸不同 |
| `MISSING` | 某轮少导了这一帧（也会先按帧数上限校验并直接失败） |

注意这是**两轮之间**的自比对，验证的是确定性；它不与仓库里的历史基线比对，
因此每次都从当前代码出发，不会因为渲染的合理改动而失败。要长期锁定某个视觉基线，
把 `build/shots/seq/` 里的 PNG 提交进仓库另行比对。

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

这些 golden 帧放在 `tests/golden/`：每个用例配一个 `.bin`（5624 字节）与一对 `.png`
（`<name>.go.png` 由本项目解码器渲染、`<name>.oracle.png` 由 Python oracle 渲染，二者逐像素比对）。

- **`.bin` 是提交的 fixture，不是构建产物**。它由 `tools/genframes/main.go` 用规范打包公式
  确定性生成，编码了各用例刻意构造的格式边界，代表"正确打包的参考帧"——测试据此验证
  两端解码一致。改格式意图要增删用例时，手动跑 `tools/genframes` 重新产出 `.bin` 即可，
  **它不是构建/测试流水线的一环**，测试不依赖它。
- **`.png` 是派生渲染，已被 `.gitignore` 忽略**，不要提交（两侧压缩级别不同，字节本就不可比）。

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

## 提交前检查（pre-commit hook）

仓库自带 `.githooks/pre-commit`，只检查**本次暂存（staged）的文件**，
工作区里遗留的旧问题不会误伤无关提交。它通过 `core.hooksPath` 随仓库分发
（不是 `.git/hooks/`——那个不入库、不跟着 clone 走），所以每个 clone 都生效。

**新 clone 后若 hook 没生效，跑一次 `.\setup.ps1` 即可（它会自动执行下面这条并体检依赖）：**

```powershell
git config core.hooksPath .githooks
```

会检查三件事，任一不通过则以**退出码 1** 拦截提交：

| 检查 | 触发条件 | 原因 |
|---|---|---|
| `build.ps1` 必须纯 ASCII | 暂存了含非 ASCII 字节的 `*.ps1` | PowerShell 5.1 把无 BOM 的 UTF-8 当 ANSI 读，一个非 ASCII 字节就能破坏引号配对，让整段脚本"伪失败"且报错信息严重偏离根因（详见脚本顶部的自检注释）。`build.ps1` 里也内置了同样的字节级自检作为第二道防线 |
| 不得提交构建产物 | 暂存了 `*.exe` / `*.dll` / `*.o` / `*.a` / `*.so` 或 `build/*` | 这些是编译/导出输出，应走 `.gitignore`，避免把二进制塞进历史 |
| Go 文件必须 gofmt 干净 | 暂存了未格式化的 `*.go` | 统一代码风格；找不到 `gofmt` 时跳过该项并提示，不会卡死 |

绕过（确认自己清楚在做什么时）：

```powershell
git commit -n     # 等价于 --no-verify，跳过所有 hook
```

> 提示：空提交（没有暂存文件）或合并提交会自动放行，无需绕过。

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

## 与官方仓库的关系（对照结论）

本项目是 **C1-Slim 的 PC 模拟器**，它本身是个新工具，但派生自两个上游，必须分清：

| 上游 | 语言/形态 | 与本项目的关系 |
|---|---|---|
| `theBillLee/c1-slim` | C 固件/启动器（`src/` + `Makefile`，GPL-3.0，默认分支 `main`） | **只借鉴**它的帧格式（5624 字节 strip-major）与刷新时序（全刷 ~700ms）；该仓 `src/` 下**没有** `c1device` 目录 |
| `App/c1device` | Go SDK（独立的 App SDK 仓库） | 本项目 `c1device/` 模块的**真正来源**：原样复制其无 tag 文件，再用 build tag 隔离出模拟器后端 |

> 一句话：固件仓给"画什么/怎么刷"的规范，Go SDK 给"怎么写应用"的代码；模拟器把两者搬到 PC 上跑。

**代码边界（已对照核实）**：

- `c1device/` 里无 build tag 的文件（`device.go` / `text.go` / `keymap.go` / `bitmap.go` 及其测试）是**原样复制**，经 grep 确认未被注入 SDL/host 代码（仅 `keymap.go` 多了一句注释）。
- 真机侧（`//go:build linux && mipsle`）与模拟器侧（`//go:build !linux || !mipsle`）严格隔离；交叉编译 `linux/mipsle` 时模拟器代码根本不参与编译，真机产物与上游一致。
- 新增的共享文件（`decode.go` 等）无 tag，两端共用。

**`.gitignore` 已与上游对齐**：`build/`、`third_party/*.dll`、`tests/golden/*.png` 覆盖构建产物；并参照固件仓补了 `__pycache__/` 与 `.DS_Store`（Python oracle 运行必生成字节码缓存）。

**可反向贡献回上游的部分**：golden 帧测试向量、帧格式文档、Python 交叉校验 oracle 这些与语言无关、且直接验证固件 5624 字节格式的内容，适合提给 `theBillLee/c1-slim`；而模拟器后端（仅 build-tag 隔离的 host 文件）若要回流，目标应是 `App/c1device`，且需 SDK 维护者同意扩大其使用范围。

---

## 许可

本项目因改造上游 `App/c1device` 而沿用 **GPL-3.0**，详见 `LICENSE` 与 `NOTICE.md`。

- 字体 `pkg-font.bin`：SIL OFL 1.1（`font-LICENSE.txt` 必须一同分发）
- `tools/oracle/frame2png.py`：来自 c1slim-toolkit，MIT
- `purego-sdl3`：Unlicense（公有领域）

---

## 未实现（后续可加）

- 视觉基线比对（把 `build/shots/seq/` 的 PNG 入库并逐帧比对，锁定视觉回归；
  现有的 `-Target regress` 只验证"两轮之间确定"，不与历史基线比对）
- 命令行参数形式（目前只用环境变量）
- 鼠标/触摸事件（C1 Max 是触屏，C1-Slim 只有键盘，故暂不需要）
