# C1-Slim / MP-D261 PC 模拟器

在 PC 上用 **SDL3** 窗口模拟快易典 C1-Slim（296×152 单色墨水屏）的显示效果。
**同一份 Go 应用源码**两端通用：

| 目标 | 命令 | 产物 |
|---|---|---|
| PC 模拟器 | `.\build.ps1 -Target sim` | Windows 窗口程序（1184×608） |
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

- 墨水屏观感：白→黑→白全刷闪烁、残影灰度、~150ms/~700ms 刷新时序
- 软件按键自动重复（真机由 evdev `value==2` 提供）
- headless 截图 / 命令行开关（放大倍数、残影强度等）
