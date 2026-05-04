# ZenStates-Linux (Go 版)

AMD Ryzen 处理器超频工具集——Go 语言改写版，带精美的终端展示效果。

## 项目结构

```
.
├── go.mod
├── Makefile
├── cmd/
│   ├── zenstates/              # P-State 管理工具 (主入口)
│   └── togglecode/             # ASUS 主板 Q-Code 切换
├── internal/
│   ├── display/                # 🎨 终端展示库 (表格/颜色/图标)
│   ├── msr/                    # MSR 寄存器读写
│   └── pstate/                 # P-State 解析与位操作
└── README.md
```

## 构建

```bash
make                    # 编译所有工具
sudo make install       # 安装到 /usr/local/bin
```

## zenstates — 动态编辑 AMD Ryzen P-State

需要 root 权限和 `msr` 内核模块：`sudo modprobe msr`

### 展示效果（十进制）

```
  AMD Ryzen P-States

┌─────────┬──────────────┬─────┬─────┬─────┬──────────┬───────────────┐
│ P-State │ Status       │ FID │ DID │ VID │ Frequency │ vCore         │
├─────────┼──────────────┼─────┼─────┼─────┼──────────┼───────────────┤
│ P0      │ ● Enabled    │ 152 │ 8   │ 32  │ 3800 MHz │ 1.35000 V     │
│ P1      │ ○ Disabled   │ --  │ --  │ --  │    --    │    --         │
│ P2      │ ● Enabled    │ 132 │ 12  │ 104 │ 3500 MHz │ 0.90000 V     │
│ P3      │ ○ Disabled   │ --  │ --  │ --  │    --    │    --         │
│ P4      │ ○ Disabled   │ --  │ --  │ --  │    --    │    --         │
│ P5      │ ○ Disabled   │ --  │ --  │ --  │    --    │    --         │
│ P6      │ ○ Disabled   │ --  │ --  │ --  │    --    │    --         │
│ P7      │ ○ Disabled   │ --  │ --  │ --  │    --    │    --         │
└─────────┴──────────────┴─────┴─────┴─────┴──────────┴───────────────┘

  C6 State
    Package  ●  Enabled
    Core     ●  Enabled
```

修改 P-State 时显示 diff 对比（十进制）：

```
  P0 Changes

┌───────────┬────────────────────┬────────────────────┐
│ Field     │ Before             │ After              │
├───────────┼────────────────────┼────────────────────┤
│ Frequency │ 3800 MHz           │ 3900 MHz           │
│ vCore     │ 1.35000 V          │ 1.30000 V          │
└───────────┴────────────────────┴────────────────────┘
```

### 完整命令行

```
  ZenStates-Linux — AMD Ryzen P-State Editor
  Dynamically edit AMD Ryzen processor P-States via MSR

  Usage: zenstates [options]

  Options:
    -l              List all P-States
    -p <0-7>        P-State to set
    --enable        Enable P-State
    --disable       Disable P-State

    --freq <MHz>    Target frequency (e.g. 3800)         ← 新! 直接设频率
    --voltage <V>   Target core voltage (e.g. 1.35)      ← 新! 直接设电压

    -f <val>        FID to set (legacy, decimal)
    -d <val>        DID to set (legacy, decimal)
    -v <val>        VID to set (legacy, decimal)

    --c6-enable     Enable C-State C6
    --c6-disable    Disable C-State C6
    --no-color      Disable colored output
    --governor <g>  Set CPU frequency governor (performance/powersave/...)

  Examples:
    zenstates -l                                # List all P-States
    zenstates -p 0 --freq 3800 --voltage 1.35   # P0: 3800MHz, 1.3500V
    zenstates -p 1 --disable                   # Disable P1
    zenstates -p 0 -f 152 -d 8 -v 32           # Legacy: FID/DID/VID
    zenstates --governor performance            # Set CPU governor
```

### 示例

```bash
# 列出当前所有 P-State
sudo ./bin/zenstates -l

# 设置 P0: 3800MHz @ 1.3500V (推荐方式)
sudo ./bin/zenstates -p 0 --freq 3800 --voltage 1.35

# 禁用 P1
sudo ./bin/zenstates -p 1 --disable

# 启用 C6
sudo ./bin/zenstates --c6-enable

# 传统方式 - 直接设 FID/DID/VID
sudo ./bin/zenstates -p 0 -f 152 -d 8 -v 32

# 设置 CPU 频率调度器为 performance
sudo ./bin/zenstates --governor performance
```

## togglecode — ASUS Q-Code 显示开关

切换 ASUS Crosshair VI Hero 等主板的 Q-Code 显示的开启/关闭。

```
  ASUS Q-Code Display Toggle
  Super I/O: 0x2E/0x2F · GPIO LD7 · Reg 0xF0

  • Entering Super I/O config mode...
  • Selecting GPIO logical device (LD7)...
  • Reading GPIO register 0xF0...
  • Current state: Q-Code = ON (0xF0 = 0x48, bit3=1)
  • Toggling bit 3...

  ✔ Q-Code display toggled successfully!

  • Q-Code: ON → OFF
  • Reg 0xF0: 0x48 → 0x40  (bit3: 1→0)
```

## 展示特性

| 特性 | 说明 |
|------|------|
| **彩色输出** | 终端自动检测，支持 ANSI 彩色 |
| **Unicode 图标** | ✔ ✘ ● ○ ▶ • 等状态图标 |
| **表格边框** | 使用 ┌─┐│└┘ 等 Unicode 画线字符 |
| **Diff 对比** | 修改 P-State 时显示 before/after 对比 |
| **直接设频率/电压** | `--freq 3800 --voltage 1.35` 自动计算 FID/DID/VID |
| **CPU 调度器** | `--governor performance` — 设置频率调度策略 |
| **禁用颜色** | `--no-color` 参数/重定向时自动降级 |
| **自动降级** | 输出到文件/管道时自动移除颜色代码 |

## 频率对照表 (DID=8)

| FID | 频率 (MHz) |
|-----|-----------|
| 144 | 3600      |
| 148 | 3700      |
| 152 | 3800      |
| 156 | 3900      |
| 160 | 4000      |
| 164 | 4100      |

## 电压对照表

| VID | 电压 (V) |
|-----|---------|
| 48  | 1.2500  |
| 40  | 1.3000  |
| 32  | 1.3500  |
| 24  | 1.4000  |
| 16  | 1.4500  |

## 与 Python 原版的区别

| 方面 | Python 版 | Go 版 |
|------|-----------|-------|
| **数值格式** | 十六进制输入/输出 | **十进制输入/输出** |
| **展示效果** | 纯文本 | **彩色表格 + Unicode 图标 + Diff 对比** |
| **MSR 读写** | `os.lseek/read/write` | `os.File.ReadAt/WriteAt` + binary |
| **位操作** | 全局 `setbits()` | `pstate.PState` 方法链 |
| **端口 I/O** | `portio` + `iopl(3)` | `/dev/port` 文件操作 |
| **CPU 调度器** | ✘ 无此功能 | **`--governor` 一键设置频率策略** |
| **部署** | 需 Python + 依赖 | **静态编译单二进制**，零依赖 |
