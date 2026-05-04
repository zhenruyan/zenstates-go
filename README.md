# ZenStates-Linux (Go Edition)

AMD Ryzen processor overclocking toolkit — rewritten in Go with beautiful terminal display.

```
.
├── go.mod
├── Makefile
├── cmd/
│   ├── zenstates/              # P-State management tool
│   └── togglecode/             # ASUS Q-Code display toggle
├── internal/
│   ├── display/                # 🎨 Terminal display (tables/colors/icons)
│   ├── msr/                    # MSR register read/write
│   └── pstate/                 # P-State parsing & bit manipulation
├── README.md                   # English documentation
├── README_zh.md                # 中文文档
└── ...
```

## Build

```bash
make                    # Build all tools
sudo make install       # Install to /usr/local/bin
```

## zenstates — AMD Ryzen P-State Editor

Requires root and the `msr` kernel module: `sudo modprobe msr`

### Output Preview

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

Diff display when modifying a P-State:

```
  P0 Changes

┌───────────┬────────────────────┬────────────────────┐
│ Field     │ Before             │ After              │
├───────────┼────────────────────┼────────────────────┤
│ Frequency │ 3800 MHz           │ 3900 MHz           │
│ vCore     │ 1.35000 V          │ 1.30000 V          │
└───────────┴────────────────────┴────────────────────┘
```

### Usage

```
  ZenStates-Linux — AMD Ryzen P-State Editor
  Dynamically edit AMD Ryzen processor P-States via MSR

  Usage: zenstates [options]

  Options:
    -l              List all P-States
    -p <0-7>        P-State to set
    --enable        Enable P-State
    --disable       Disable P-State

    --freq <MHz>    Target frequency (e.g. 3800)         ← NEW! direct freq setting
    --voltage <V>   Target core voltage (e.g. 1.35)      ← NEW! direct voltage setting

    -f <val>        FID to set (legacy, decimal)
    -d <val>        DID to set (legacy, decimal)
    -v <val>        VID to set (legacy, decimal)

    --c6-enable     Enable C-State C6
    --c6-disable    Disable C-State C6
    --no-color      Disable colored output

  Examples:
    zenstates -l                                # List all P-States
    zenstates -p 0 --freq 3800 --voltage 1.35   # P0: 3800MHz, 1.3500V
    zenstates -p 1 --disable                   # Disable P1
    zenstates -p 0 -f 152 -d 8 -v 32           # Legacy: FID/DID/VID
```

### Examples

```bash
# List all P-States
sudo ./bin/zenstates -l

# Set P0 to 3800MHz @ 1.3500V (recommended)
sudo ./bin/zenstates -p 0 --freq 3800 --voltage 1.35

# Disable P1
sudo ./bin/zenstates -p 1 --disable

# Enable C6
sudo ./bin/zenstates --c6-enable

# Legacy: set FID/DID/VID directly
sudo ./bin/zenstates -p 0 -f 152 -d 8 -v 32
```

## togglecode — ASUS Q-Code Display Toggle

Toggle the Q-Code display on ASUS Crosshair VI Hero and other boards with a compatible Super I/O chip.

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

## Display Features

| Feature | Description |
|---------|------------|
| **Color output** | Auto-detects terminal, ANSI colors |
| **Unicode icons** | ✔ ✘ ● ○ ▶ • status indicators |
| **Box-drawing tables** | ┌─┐│└┘ Unicode table borders |
| **Diff view** | Before/after comparison when modifying P-States |
| **Direct freq/voltage** | `--freq 3800 --voltage 1.35` auto-calculates FID/DID/VID |
| **No-color mode** | `--no-color` flag, auto-disable on pipe/redirect |
| **Auto fallback** | Removes ANSI codes when output is piped to a file |

## Frequency Table (DID=8)

| FID | Frequency (MHz) |
|-----|----------------|
| 144 | 3600           |
| 148 | 3700           |
| 152 | 3800           |
| 156 | 3900           |
| 160 | 4000           |
| 164 | 4100           |

## Voltage Table

| VID | Voltage (V) |
|-----|------------|
| 48  | 1.2500     |
| 40  | 1.3000     |
| 32  | 1.3500     |
| 24  | 1.4000     |
| 16  | 1.4500     |

## Differences from Python Original

| Aspect | Python Version | Go Version |
|--------|---------------|------------|
| **Number format** | Hexadecimal I/O | **Decimal I/O** |
| **Display** | Plain text | **Color table + Unicode icons + Diff view** |
| **High-level API** | — | **`--freq` / `--voltage`** auto calculation |
| **MSR I/O** | `os.lseek/read/write` | `os.File.ReadAt/WriteAt` + binary |
| **Bit ops** | Global `setbits()` | `pstate.PState` method chain |
| **Port I/O** | `portio` + `iopl(3)` | `/dev/port` file operations |
| **Deployment** | Python + dependencies | **Static single binary**, zero dependencies |

## Credits

This Go port is based on the original **[ZenStates-Linux](https://github.com/r4m0n/ZenStates-Linux)** Python project by [r4m0n](https://github.com/r4m0n).

- `zenstates.py` — Original P-State editing tool
- `togglecode.py` — Original ASUS Q-Code toggle script

Thank you for the foundational work on AMD Ryzen overclocking utilities!
