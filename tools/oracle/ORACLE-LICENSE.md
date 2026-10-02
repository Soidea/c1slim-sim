# 第三方 oracle 说明

## frame2png.py

- **来源**：`mason-yb-zhang/c1slim-toolkit`，路径 `tools/frame2png.py`
- **许可**：该仓库 README 声明"本仓库新增源码（radio、c1apprun、c1dec/decoder.c、**tools**）：**MIT**"
- **用途**：作为**独立第三方**的帧解码实现，用于交叉校验本项目的 `c1device.DecodeGray`。
  它是除上游 `text.go` 之外、由不同作者独立写出的同一份格式实现，因此是有效的 oracle。
- **未改动**：原样复制，未做任何修改。

## pngdiff.py

- **归属**：本项目新增
- **许可**：GPL-3.0（与项目整体一致）
- **用途**：逐像素比对 `*.oracle.png`（Python 产出）与 `*.go.png`（本项目产出）
- **依赖**：Pillow、numpy

## 为什么必须逐像素比，不能比字节

两侧的 PNG 编码器不同（Python 用 `zlib.compress(raw, 9)`，Go 用 `image/png` 默认级别），
即使像素完全相同，压缩后的字节流也必然不同。因此比对必须解回像素后再比。
