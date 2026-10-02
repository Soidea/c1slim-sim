# NOTICE —— 来源与改动清单

## 1. 上游溯源

| 项 | 值 |
|---|---|
| 上游仓库 | `https://github.com/fwz233-RE/C1auncher.git` |
| commit | `3684f450a6b0193c4f3929d1c85795abfbc49c1c` |
| 取用路径 | `App/c1device/`（整体复制为本项目的 `c1device/`） |
| 上游许可 | **GPL-3.0**（上游根 `LICENSE` 明文覆盖 `App/` 目录） |
| 本项目许可 | **GPL-3.0**（派生自上述代码，见 `LICENSE`） |

上游 `LICENSE` 原文节选：

> C1-Slim device-side source licensing … first-party source code in
> `C1ancher/`, `C1ancher-server/`, `App/`, `Pinao/`, `ChiChuGames/`, `examples/` …

首次提交记录了该来源：`import: App/c1device @3684f45 from fwz233-RE/C1auncher (GPL-3.0)`

---

## 2. 逐文件改动清单

### 原样复制，未改动（禁止改动 —— 将来对接真机的唯一入口）

| 文件 | 说明 |
|---|---|
| `c1device/device.go` | `Frame` / `Key` / `Event` / `Platform` 定义 |
| `c1device/platform_device_linux_mipsle.go` | 真机后端（tag `linux && mipsle`） |
| `c1device/text.go`、`bitmap.go`、`keymap.go` | 画布、点阵字体、真机 evdev 键位映射 |
| `c1device/desktop_linux.go`、`desktop_stub.go` | `ReturnToDesktop()` |
| `c1device/*_test.go` | 上游测试 |

### 重写

| 文件 | 改动 |
|---|---|
| `c1device/platform_host.go` | 由"仅存帧不渲染、事件通道永不写入"的空壳，重写为 SDL3 窗口后端：帧去重、首帧强制全刷、独立 pump goroutine（`LockOSThread` 独占 SDL 调用）、`UpdateTexture` + `RenderTexture` 上屏、关窗等价于按返回键 |

### 新增

| 文件 | 说明 |
|---|---|
| `c1device/decode.go` | 1bpp 帧解码（`Frame.Pixel`、`DecodeGray`），**无 build tag，两端共用** |
| `c1device/decode_test.go` | 解码单元测试（含 strip 边界、位序、越界） |
| `c1device/keymap_host.go` | SDL 物理键 → `Event` 映射 |
| `apps/demo/*` | 示例应用 / 新应用开发模板 |
| `tools/*` | golden 生成、Go 版 frame2png、Python oracle 与比对脚本 |
| `build.ps1`、`crosscheck.ps1` | 双端构建与校验脚本 |

---

## 3. 第三方组件

| 组件 | 许可 | 说明 |
|---|---|---|
| `apps/demo/assets/pkg-font.bin` | **SIL OFL 1.1** | 取自上游 `App/book-reader/assets`，C1BF 点阵（含中文）。`font-LICENSE.txt` 必须一同分发 |
| `tools/oracle/frame2png.py` | **MIT** | 取自 `mason-yb-zhang/c1slim-toolkit` 的 `tools/`；原样复制，用作独立 oracle。详见 `tools/oracle/ORACLE-LICENSE.md` |
| `github.com/jupiterrider/purego-sdl3` | **Unlicense**（公有领域） | cgo-free 的 SDL3 绑定，运行时动态加载 `SDL3.dll` |
| `github.com/ebitengine/purego` | (间接依赖) | purego-sdl3 的底层 FFI |
| SDL3 3.4.16 | zlib | 运行时共享库，**不随仓库分发**，由 `build.ps1` 从本机复制 |

字体与 oracle 脚本的许可彼此独立，不因放入本仓库而改变；OFL 与 MIT 均与 GPL-3.0 兼容。

---

## 4. 验证声明

- ✅ PC 模拟器：窗口渲染、键鼠事件、交叉编译均通过
- ✅ 帧解码：与第三方 Python oracle **逐像素一致**（13/13）
- ✅ 真机产物：`ELF32 / 小端 / EXEC / EM_MIPS`，device 依赖不含 SDL
- ❌ **未做实机验收**：当前没有实体 C1-Slim 设备，全部结论仅限于模拟器与交叉编译层面
