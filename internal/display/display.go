// Package display provides terminal-friendly output formatting utilities.
// Uses ANSI colors when stdout is a terminal, falls back to plain text otherwise.
package display

import (
	"fmt"
	"os"
	"strings"
)

// ── Terminal detection ─────────────────────────────────────────────

var isTerminal = false

func init() {
	detectTerminal()
}

func detectTerminal() {
	stat, err := os.Stdout.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
		isTerminal = true
	}
}

// DisableColor forces plain text output (no ANSI codes).
func DisableColor() {
	isTerminal = false
}

// ── ANSI color codes ───────────────────────────────────────────────

type Color string

const (
	Reset   Color = "\033[0m"
	Bold    Color = "\033[1m"
	Dim     Color = "\033[2m"

	Red     Color = "\033[31m"
	Green   Color = "\033[32m"
	Yellow  Color = "\033[33m"
	Blue    Color = "\033[34m"
	Magenta Color = "\033[35m"
	Cyan    Color = "\033[36m"
	White   Color = "\033[37m"

	BgRed    Color = "\033[41m"
	BgGreen  Color = "\033[42m"
	BgYellow Color = "\033[43m"
	BgBlue   Color = "\033[44m"
)

// colorize wraps text in color if terminal output is active.
func colorize(c Color, s string) string {
	if !isTerminal {
		return s
	}
	return string(c) + s + string(Reset)
}

// ── Public styled output helpers ───────────────────────────────────

func Sprintf(c Color, format string, a ...any) string {
	return colorize(c, fmt.Sprintf(format, a...))
}

func BoldText(s string) string     { return colorize(Bold, s) }
func GreenText(s string) string    { return colorize(Green, s) }
func RedText(s string) string      { return colorize(Red, s) }
func YellowText(s string) string   { return colorize(Yellow, s) }
func CyanText(s string) string     { return colorize(Cyan, s) }
func BlueText(s string) string     { return colorize(Blue, s) }
func DimText(s string) string      { return colorize(Dim, s) }
func MagentaText(s string) string  { return colorize(Magenta, s) }

// ── Status icons (Unicode) ─────────────────────────────────────────

const (
	IconCheck    = "✔"
	IconCross    = "✘"
	IconArrow    = "▶"
	IconBullet   = "•"
	IconEnabled  = "●"
	IconDisabled = "○"
	IconDiff     = "→"
	IconStar     = "★"
)

// StatusText returns a colored status string with icon.
func StatusOK(text string) string {
	return GreenText(IconCheck + " " + text)
}

func StatusFail(text string) string {
	return RedText(IconCross + " " + text)
}

func StatusInfo(text string) string {
	return CyanText(IconBullet + " " + text)
}

func StatusWarn(text string) string {
	return YellowText(IconBullet + " " + text)
}

// Bullet returns a styled bullet character for list items.
func Bullet() string {
	return colorize(Cyan, IconBullet)
}

// ── Table formatting ───────────────────────────────────────────────

const (
	boxH  = "─"
	boxV  = "│"
	boxTL = "┌"
	boxTM = "┬"
	boxTR = "┐"
	boxML = "├"
	boxMM = "┼"
	boxMR = "┤"
	boxBL = "└"
	boxBM = "┴"
	boxBR = "┘"
	boxEq = "═"
)

// Table is a simple border-drawn table.
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	t := &Table{headers: headers}
	t.widths = make([]int, len(headers))
	for i, h := range headers {
		t.widths[i] = len(stripANSI(h))
	}
	return t
}

// AddRow adds a row of data to the table.
func (t *Table) AddRow(cols ...string) {
	if len(cols) != len(t.headers) {
		// Pad or truncate to match header count
		padded := make([]string, len(t.headers))
		for i := range padded {
			if i < len(cols) {
				padded[i] = cols[i]
			}
		}
		cols = padded
	}
	t.rows = append(t.rows, cols)
	for i, c := range cols {
		w := len(stripANSI(c))
		if w > t.widths[i] {
			t.widths[i] = w
		}
	}
}

// Render returns the full table as a string.
func (t *Table) Render() string {
	var sb strings.Builder

	// Top border
	t.border(&sb, boxTL, boxTM, boxTR)
	// Header
	t.dataRow(&sb, t.headers, BoldText)
	// Separator
	t.border(&sb, boxML, boxMM, boxMR)
	// Data rows
	nop := func(s string) string { return s }
	for _, row := range t.rows {
		t.dataRow(&sb, row, nop)
	}
	// Bottom border
	t.border(&sb, boxBL, boxBM, boxBR)

	return sb.String()
}

func (t *Table) border(sb *strings.Builder, left, mid, right string) {
	sb.WriteString(left)
	for i, w := range t.widths {
		sb.WriteString(strings.Repeat(boxH, w+2))
		if i < len(t.widths)-1 {
			sb.WriteString(mid)
		}
	}
	sb.WriteString(right)
	sb.WriteByte('\n')
}

func (t *Table) dataRow(sb *strings.Builder, cols []string, styler func(string) string) {
	sb.WriteString(boxV)
	for i, c := range cols {
		styled := c
		if styler != nil {
			styled = styler(c)
		}
		w := t.widths[i]
		rawLen := len(stripANSI(styled))
		pad := w - rawLen
		sb.WriteString(" " + styled + strings.Repeat(" ", pad) + " ")
		sb.WriteString(boxV)
	}
	sb.WriteByte('\n')
}

// stripANSI removes ANSI escape codes from a string for width calculation.
func stripANSI(s string) string {
	var out strings.Builder
	in := false
	for _, r := range s {
		if r == '\033' {
			in = true
		}
		if !in {
			out.WriteRune(r)
		}
		if in && r == 'm' {
			in = false
		}
	}
	return out.String()
}

// ── Progress bar ───────────────────────────────────────────────────

// ProgressBar renders a simple text progress bar.
// width: total characters, pct: 0.0-1.0
func ProgressBar(width int, pct float64, label ...string) string {
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	s := fmt.Sprintf("[%s] %3.0f%%", bar, pct*100)
	if len(label) > 0 && label[0] != "" {
		s = label[0] + " " + s
	}
	return s
}

// ── Key-Value pair formatting ──────────────────────────────────────

// KV renders a colored key=value pair.
func KV(key, value string, valColor Color) string {
	return DimText(key+"=") + colorize(valColor, value)
}

// Section renders a section header line.
func Section(title string) string {
	line := strings.Repeat(boxEq, 4)
	return fmt.Sprintf("\n %s %s %s\n", line, BoldText(title), line)
}

// Divider renders a divider line.
func Divider() string {
	if isTerminal {
		return colorize(Dim, strings.Repeat("─", 50))
	}
	return strings.Repeat("-", 50)
}
