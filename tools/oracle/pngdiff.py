#!/usr/bin/env python3
"""逐像素比对 Go 解码器与 Python oracle 的输出。

用法: pngdiff.py <golden目录>
    目录下需成对存在 <name>.oracle.png 与 <name>.go.png

注意: 只能逐像素比, 不能比 PNG 字节 —— 两侧 zlib 压缩级别不同, 字节必然不等。
"""
import sys
import glob
import os

import numpy as np
from PIL import Image

def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    d = sys.argv[1]
    pairs = sorted(glob.glob(os.path.join(d, '*.oracle.png')))
    if not pairs:
        print(f'没有找到 *.oracle.png (目录: {d})')
        return 2

    bad = 0
    for oracle in pairs:
        mine = oracle.replace('.oracle.png', '.go.png')
        name = os.path.basename(oracle).replace('.oracle.png', '')
        if not os.path.exists(mine):
            print(f'MISSING {name}: 缺少 {os.path.basename(mine)}')
            bad += 1
            continue
        a = np.array(Image.open(oracle).convert('L'), dtype=np.int16)
        b = np.array(Image.open(mine).convert('L'), dtype=np.int16)
        if a.shape != b.shape:
            print(f'FAIL    {name:14s} 尺寸不一致 {a.shape} vs {b.shape}')
            bad += 1
            continue
        diff = np.abs(a - b)
        n = int((diff != 0).sum())
        if n == 0:
            print(f'OK      {name:14s} {a.shape[1]}x{a.shape[0]}  完全一致')
        else:
            print(f'FAIL    {name:14s} {a.shape[1]}x{a.shape[0]}  不一致像素={n} 最大差={int(diff.max())}')
            bad += 1

    print(f'\n用例总数: {len(pairs)}   FAILED: {bad}')
    return 1 if bad else 0

if __name__ == '__main__':
    sys.exit(main())
