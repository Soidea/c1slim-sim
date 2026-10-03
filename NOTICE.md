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

帧格式（5624 字节 strip-major）随 `App/c1device` 一并取自上述上游。刷新时序（全刷 ~700ms、写入 ~150ms）为**借鉴** C1-Slim 固件实测值（`theBillLee/c1-slim`），未复制其代码。

---

## 2. 逐文件改动清单

### 原样复制，未改动（真机对接的唯一入口，勿改）

| 文件 | 说明 |
|---|---|
| `c1device/device.go` | `Frame` / `Key` / `Event` / `Platform` 定义 |
| `c1device/platform_device_linux_mipsle.go` | 真机后端（tag `linux && mipsle`） |
| `c1device/text.go`、`bitmap.go`、`keymap.go` | 画布、点阵字体、真机 evdev 键位映射 |
| `c1device/bitmap_test.go`、`keymap_test.go`、`text_test.go` | 上游测试（原样） |
| `c1device/input_lifecycle_linux_mipsle_test.go`、`visual_key_linux_mipsle_test.go` | 上游真机侧测试（tag `linux && mipsle`） |

> 这部分与上游 `App/c1device` 逐字节一致，可用 `diff` 核验；差异只出现在下面的 host 侧文件。

### 修改（保持上游文件名，仅动 host 侧）

| 文件 | 改动 |
|---|---|
| `c1device/platform_host.go` | 由"仅存帧不渲染、事件通道永不写入"的空壳，重写为 **SDL-free 公共骨架**：环境变量选项、PNG 导出辅助、`Draw`/`Events`/`Close`、与真机一致的帧去重 + 首帧强制全刷语义。原 SDL3 引擎移至 `platform_window.go` |
| `c1device/desktop_linux.go` | build tag `linux` → `linux && mipsle`：原实现在 `C1_C1ANCHER_TERMINAL=1` 时 `SIGKILL` 父进程，收窄后仅真机触发，避免在 Linux 桌面跑模拟器时误杀 shell |
| `c1device/desktop_stub.go` | build tag `!linux` → `!linux || !mipsle`（与上面对称，空实现覆盖所有非真机 host） |
| `c1device/go.mod`、`go.sum` | 新增 `purego` + `purego-sdl3` 依赖（SDL3 运行时动态加载，零 cgo） |

### 新增

| 文件 | 说明 |
|---|---|
| `c1device/platform_window.go` | SDL3 窗口 / 渲染 / 事件泵：独立 pump goroutine（`LockOSThread` 独占 SDL 调用）、`UpdateTexture` + `RenderTexture` 上屏、关窗等价于按返回键、放大倍数调节、按键长按自动重复、全刷闪烁与残影观感还原（默认构建，tag `!headless`） |
| `c1device/platform_headless.go` | 纯无头导出泵，**完全不链接 SDL3**（`-tags headless`），可在无 SDL3 / 无显卡的 CI 上做渲染回归 |
| `c1device/keymap_host.go` | SDL-free 的数值 scancode→rune 映射 + 长按重复配置 |
| `c1device/keymap_window.go` | SDL 类型 scancode → `Event` 翻译（默认构建，tag `!headless`） |
| `c1device/decode.go` | 1bpp 帧解码（`Frame.Pixel`、`DecodeGray`），**无 build tag，两端共用** |
| `c1device/decode_test.go` | 解码单元测试（含 strip 边界、位序、越界） |
| `c1device/platform_host_test.go`、`keymap_host_test.go` | 缩放键钳制、环境变量回落、可重复键策略、残影与闪烁的纯函数测试 |
| `apps/demo/*` | 示例应用 / 新应用开发模板 |
| `tools/*` | golden 生成、Go 版 frame2png、Python oracle 与比对脚本 |
| `build.ps1`、`crosscheck.ps1` | 双端构建与校验脚本（含 `-Target headless` 产出零 SDL 依赖二进制） |

### host 变体与 build tag 对称

| 构建 | 编译的 host 文件 | 需要 SDL3 |
|---|---|---|
| 默认 | `platform_host.go` + `platform_window.go` + `keymap_window.go` | 是（运行时 purego 动态加载） |
| `-tags headless` | `platform_host.go` + `platform_headless.go` + `keymap_host.go` | **否** |
| `linux/mipsle` | 不编译任何 host 文件 | 否 |

窗口与无头变体由互斥的 `!headless` / `headless` tag 二选一，保证 headless 构建零 SDL 依赖。

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
- ✅ headless-only 构建（`-tags headless`）：零 SDL 依赖，冒烟导出通过（`build.ps1 -Target headless`）
- ❌ **未做实机验收**：当前没有实体 C1-Slim 设备，全部结论仅限于模拟器与交叉编译层面
