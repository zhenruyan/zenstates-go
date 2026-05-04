// Package pstate implements AMD Ryzen P-State register parsing and manipulation.
//
// P-State MSR layout (64-bit):
//   Bit 63    : P-State enable (1 = enabled, 0 = disabled)
//   Bits 7:0  : FID (Frequency ID)
//   Bits 13:8 : DID (Divisor ID)
//   Bits 21:14: VID (Voltage ID)
//
// Formulas:
//   Ratio = 25 * FID / (12.5 * DID)
//   vCore = 1.55 - 0.00625 * VID
package pstate

import (
	"fmt"
	"math"
	"strings"
)

// PState represents a single P-State register value.
type PState uint64

// PStateMSRs is the list of P-State MSR addresses (P0..P7).
var PStateMSRs = []uint32{0xC0010064, 0xC0010065, 0xC0010066, 0xC0010067,
	0xC0010068, 0xC0010069, 0xC001006A, 0xC001006B}

// PStateCount returns the number of P-States (8).
func PStateCount() int { return len(PStateMSRs) }

// Enabled returns true if this P-State is enabled (bit 63 set).
func (p PState) Enabled() bool {
	return p&(1<<63) != 0
}

// FID returns the Frequency ID (bits 7:0).
func (p PState) FID() uint64 {
	return uint64(p) & 0xFF
}

// DID returns the Divisor ID (bits 13:8).
func (p PState) DID() uint64 {
	return (uint64(p) >> 8) & 0x3F
}

// VID returns the Voltage ID (bits 21:14).
func (p PState) VID() uint64 {
	return (uint64(p) >> 14) & 0xFF
}

// Ratio returns the frequency ratio: 25 * FID / (12.5 * DID).
func (p PState) Ratio() float64 {
	if p.DID() == 0 {
		return 0
	}
	return 25.0 * float64(p.FID()) / (12.5 * float64(p.DID()))
}

// FrequencyMHz returns the approximate frequency in MHz (Ratio * 100).
func (p PState) FrequencyMHz() float64 {
	return p.Ratio() * 100
}

// VCore returns the core voltage: 1.55 - 0.00625 * VID.
func (p PState) VCore() float64 {
	return 1.55 - 0.00625*float64(p.VID())
}

// String returns a compact single-line representation.
func (p PState) String() string {
	if !p.Enabled() {
		return "Disabled"
	}
	return fmt.Sprintf("FID=%d DID=%d VID=%d  %5.0fMHz  %.5fV",
		p.FID(), p.DID(), p.VID(), p.FrequencyMHz(), p.VCore())
}

// Formatted returns a multi-line, human-friendly representation.
func (p PState) Formatted() string {
	if !p.Enabled() {
		return "Disabled"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  FID  : %d\n", p.FID()))
	b.WriteString(fmt.Sprintf("  DID  : %d\n", p.DID()))
	b.WriteString(fmt.Sprintf("  VID  : %d\n", p.VID()))
	b.WriteString(fmt.Sprintf("  Freq : %.0f MHz\n", p.FrequencyMHz()))
	b.WriteString(fmt.Sprintf("  Ratio: %.2f\n", p.Ratio()))
	b.WriteString(fmt.Sprintf("  vCore: %.5f V\n", p.VCore()))
	return b.String()
}

// ShortStatus returns a one-line status string with colored indicators.
func (p PState) ShortStatus() string {
	if !p.Enabled() {
		return fmt.Sprintf("%-9s  %4s  %4s  %4s  %7s  %8s",
			"Disabled", "--", "--", "--", "--MHz", "--.--V")
	}
	return fmt.Sprintf("%-9s  %3d  %3d  %3d  %5.0fMHz  %.5fV",
		"Enabled", p.FID(), p.DID(), p.VID(), p.FrequencyMHz(), p.VCore())
}

// Diff returns a visual diff string showing what changed between old and new.
func Diff(old, new PState) string {
	var parts []string

	if old.Enabled() != new.Enabled() {
		if new.Enabled() {
			parts = append(parts, "Enabled")
		} else {
			parts = append(parts, "Disabled")
		}
	}
	if old.FID() != new.FID() {
		parts = append(parts, fmt.Sprintf("FID %d→%d", old.FID(), new.FID()))
	}
	if old.DID() != new.DID() {
		parts = append(parts, fmt.Sprintf("DID %d→%d", old.DID(), new.DID()))
	}
	if old.VID() != new.VID() {
		parts = append(parts, fmt.Sprintf("VID %d→%d", old.VID(), new.VID()))
	}

	if len(parts) == 0 {
		return "no changes"
	}
	return strings.Join(parts, ", ")
}

// SetBits sets 'length' bits starting at 'base' position to 'new' value.
func SetBits(val uint64, base, length int, new uint64) uint64 {
	mask := (uint64(1)<<length - 1) << base
	return (val & ^mask) | (new<<base)&mask
}

// SetEnable sets or clears the enable bit (bit 63).
func (p PState) SetEnable(enabled bool) PState {
	if enabled {
		return PState(SetBits(uint64(p), 63, 1, 1))
	}
	return PState(SetBits(uint64(p), 63, 1, 0))
}

// SetFID sets the FID field (bits 7:0).
func (p PState) SetFID(fid uint64) PState {
	return PState(SetBits(uint64(p), 0, 8, fid))
}

// SetDID sets the DID field (bits 13:8).
func (p PState) SetDID(did uint64) PState {
	return PState(SetBits(uint64(p), 8, 6, did))
}

// SetVID sets the VID field (bits 21:14).
func (p PState) SetVID(vid uint64) PState {
	return PState(SetBits(uint64(p), 14, 8, vid))
}

// ── C6 State ───────────────────────────────────────────────────────

// C6State represents the C6 package/core state.
type C6State struct {
	PackageEnabled bool
	CoreEnabled    bool
}

// C6PackageMSR is the MSR address for C6 package state.
const C6PackageMSR = 0xC0010292

// C6CoreMSR is the MSR address for C6 core state.
const C6CoreMSR = 0xC0010296

// C6PackageBit is the bit position for C6 package enable.
const C6PackageBit = 32

// C6CoreBits are the bit positions for C6 core enable.
var C6CoreBits = [3]int{6, 14, 22}

// ParseC6StateFromMSR parses C6 state from raw MSR values.
func ParseC6StateFromMSR(pkgVal, coreVal uint64) C6State {
	coreEnabled := true
	for _, b := range C6CoreBits {
		if coreVal&(1<<b) == 0 {
			coreEnabled = false
			break
		}
	}
	return C6State{
		PackageEnabled: pkgVal&(1<<C6PackageBit) != 0,
		CoreEnabled:    coreEnabled,
	}
}

// C6PackageMask returns the bitmask for C6 package enable.
func C6PackageMask() uint64 {
	return 1 << C6PackageBit
}

// C6CoreMask returns the combined bitmask for C6 core enable.
func C6CoreMask() uint64 {
	var mask uint64
	for _, b := range C6CoreBits {
		mask |= 1 << b
	}
	return mask
}

// String returns a compact display of C6 state.
func (c C6State) String() string {
	pkg := "Disabled"
	core := "Disabled"
	if c.PackageEnabled {
		pkg = "Enabled"
	}
	if c.CoreEnabled {
		core = "Enabled"
	}
	return fmt.Sprintf("Package: %s | Core: %s", pkg, core)
}

// Formatted returns a multi-line C6 state display.
func (c C6State) Formatted() string {
	var b strings.Builder
	b.WriteString("  C6 State\n")
	if c.PackageEnabled {
		b.WriteString(fmt.Sprintf("    Package: %s\n", c.pkgLabel()))
	} else {
		b.WriteString(fmt.Sprintf("    Package: %s\n", c.pkgLabel()))
	}
	if c.CoreEnabled {
		b.WriteString(fmt.Sprintf("    Core   : %s\n", c.coreLabel()))
	} else {
		b.WriteString(fmt.Sprintf("    Core   : %s\n", c.coreLabel()))
	}
	return b.String()
}

func (c C6State) pkgLabel() string {
	if c.PackageEnabled {
		return "● Enabled"
	}
	return "○ Disabled"
}

func (c C6State) coreLabel() string {
	if c.CoreEnabled {
		return "● Enabled"
	}
	return "○ Disabled"
}

// ── Tables ─────────────────────────────────────────────────────────

// FormatRatioTable returns a formatted table of FID values and frequencies.
func FormatRatioTable() string {
	var sb strings.Builder
	sb.WriteString("┌──────┬──────────┐\n")
	sb.WriteString("│ FID  │ Freq(MHz)│\n")
	sb.WriteString("├──────┼──────────┤\n")
	for fid := uint64(144); fid <= 164; fid++ {
		freq := math.Round(25.0 * float64(fid) / 12.5)
		sb.WriteString(fmt.Sprintf("│ %3d  │ %6.0f   │\n", fid, freq))
	}
	sb.WriteString("└──────┴──────────┘\n")
	return sb.String()
}

// FormatVoltageTable returns a formatted table of VID values and voltages.
func FormatVoltageTable() string {
	var sb strings.Builder
	sb.WriteString("┌──────┬──────────┐\n")
	sb.WriteString("│ VID  │ Voltage  │\n")
	sb.WriteString("├──────┼──────────┤\n")
	for vid := uint64(48); vid >= 16; vid-- {
		vcore := 1.55 - 0.00625*float64(vid)
		sb.WriteString(fmt.Sprintf("│ %3d  │ %.4fV  │\n", vid, vcore))
	}
	sb.WriteString("└──────┴──────────┘\n")
	return sb.String()
}

// ── Frequency/Voltage conversion ───────────────────────────────────

// DefaultDID is the standard divisor used when converting frequency to FID.
const DefaultDID = 8

// FreqToFID calculates the FID value for a target frequency (MHz) at the given DID.
// Formula: freq = 200 * FID / DID  →  FID = freq * DID / 200
func FreqToFID(freqMHz, did uint64) uint64 {
	return freqMHz * did / 200
}

// FIDToFreq calculates the approximate frequency (MHz) from FID and DID.
func FIDToFreq(fid, did uint64) float64 {
	if did == 0 {
		return 0
	}
	return 200.0 * float64(fid) / float64(did)
}

// VoltageToVID calculates the VID value for a target core voltage (V).
// Formula: vCore = 1.55 - 0.00625 * VID  →  VID = (1.55 - vCore) / 0.00625
func VoltageToVID(vcore float64) uint64 {
	vid := (1.55 - vcore) / 0.00625
	if vid < 0 {
		return 0
	}
	if vid > 255 {
		return 255
	}
	return uint64(math.Round(vid))
}

// ── Parsing ────────────────────────────────────────────────────────

// ParseUint parses a decimal string to uint64.
func ParseUint(s string) (uint64, error) {
	var val uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid decimal character %q", c)
		}
		val = val*10 + uint64(c-'0')
	}
	return val, nil
}
