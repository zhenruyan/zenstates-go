// Command zenstates dynamically edits AMD Ryzen processor P-States.
//
// It reads/writes MSR registers via /dev/cpu/N/msr (requires root and modprobe msr).
//
// Usage:
//
//	zenstates -l                                          # List all P-States
//	zenstates -p 0 --freq 3800 --voltage 1.35             # Set P0: 3800MHz @ 1.3500V
//	zenstates -p 1 --disable                              # Disable P1
//	zenstates -p 2 --enable                               # Enable P2
//	zenstates --c6-enable                                 # Enable C6 state
//	zenstates --c6-disable                                # Disable C6 state
//	zenstates -p 0 -f 152 -d 8 -v 32                      # Legacy: set FID/DID/VID directly
//	zenstates --governor performance                    # Set CPU governor to performance
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/amdcpu-overclocking/internal/display"
	"github.com/amdcpu-overclocking/internal/msr"
	"github.com/amdcpu-overclocking/internal/pstate"
)

func main() {
	// ── CLI flags ────────────────────────────────────────────────
	list := flag.Bool("l", false, "List all P-States")
	pstateIdx := flag.Int("p", -1, "P-State to set (0-7)")
	enable := flag.Bool("enable", false, "Enable P-State")
	disable := flag.Bool("disable", false, "Disable P-State")
	fidStr := flag.String("f", "", "FID to set")
	didStr := flag.String("d", "", "DID to set")
	vidStr := flag.String("v", "", "VID to set")
	freqMHz := flag.Uint64("freq", 0, "Target frequency in MHz (e.g. 3800)")
	vcore := flag.Float64("voltage", 0, "Target core voltage (e.g. 1.35)")
	c6Enable := flag.Bool("c6-enable", false, "Enable C-State C6")
	c6Disable := flag.Bool("c6-disable", false, "Disable C-State C6")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	governor := flag.String("governor", "", "Set CPU frequency governor (e.g. performance)")

	flag.Parse()

	if *noColor {
		display.DisableColor()
	}

	// Validate p-state index
	if *pstateIdx != -1 && (*pstateIdx < 0 || *pstateIdx > 7) {
		fmt.Fprintf(os.Stderr, "%s P-State must be between 0 and 7, got %d\n",
			display.RedText("Error:"), *pstateIdx)
		os.Exit(1)
	}

	// ── -l: List all P-States ───────────────────────────────────
	if *list {
		printPStateTable()
		printC6Status()

		// If only -l was specified, we're done
		if *pstateIdx == -1 && !*c6Enable && !*c6Disable {
			return
		}
	}

	// ── -p N: Modify a specific P-State ─────────────────────────
	if *pstateIdx >= 0 {
		modifyPState(*pstateIdx, *enable, *disable, *fidStr, *didStr, *vidStr, *freqMHz, *vcore)
	}

	// ── --c6-enable / --c6-disable ──────────────────────────────
	if *c6Enable {
		modifyC6(true)
	}
	if *c6Disable {
		modifyC6(false)
	}

	// ── --governor ───────────────────────────────────────────
	if *governor != "" {
		setGovernor(*governor)
	}

	// ── No action shown? Show help ──────────────────────────────
	if !*list && *pstateIdx == -1 && !*c6Enable && !*c6Disable && *governor == "" {
		flag.Usage()
	}
}

// ── P-State Table ────────────────────────────────────────────────

func printPStateTable() {
	fmt.Println()
	fmt.Printf("  %s\n", display.BoldText("AMD Ryzen P-States"))
	fmt.Println()

	// Read all P-States
	states := make([]pstate.PState, pstate.PStateCount())
	for i := 0; i < len(states); i++ {
		val, err := msr.ReadMSR(pstate.PStateMSRs[i], 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s reading P%d: %v\n",
				display.RedText("Error:"), i, err)
			os.Exit(1)
		}
		states[i] = pstate.PState(val)
	}

	// Build table
	tbl := display.NewTable("P-State", "Status", "FID", "DID", "VID", "Frequency", "vCore")

	for i, s := range states {
		id := fmt.Sprintf("P%d", i)

		var status string
		if s.Enabled() {
			status = display.GreenText(display.IconEnabled + " Enabled")
		} else {
			status = display.RedText(display.IconDisabled + " Disabled")
		}

		var fid, did, vid string
		if s.Enabled() {
			fid = fmt.Sprintf("%d", s.FID())
			did = fmt.Sprintf("%d", s.DID())
			vid = fmt.Sprintf("%d", s.VID())
		} else {
			fid = display.DimText("--")
			did = display.DimText("--")
			vid = display.DimText("--")
		}

		var freq string
		if s.Enabled() {
			freq = fmt.Sprintf("%5.0f MHz", s.FrequencyMHz())
		} else {
			freq = display.DimText("   --   ")
		}

		var vcore string
		if s.Enabled() {
			vcore = fmt.Sprintf("%.5f V", s.VCore())
		} else {
			vcore = display.DimText("  --  ")
		}

		tbl.AddRow(id, status, fid, did, vid, freq, vcore)
	}

	fmt.Println(tbl.Render())
}

// ── C6 Status ────────────────────────────────────────────────────

func printC6Status() {
	pkgVal, err := msr.ReadMSR(pstate.C6PackageMSR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s reading C6 package MSR: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}
	coreVal, err := msr.ReadMSR(pstate.C6CoreMSR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s reading C6 core MSR: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}

	c6 := pstate.ParseC6StateFromMSR(pkgVal, coreVal)

	pkgIcon := display.IconDisabled
	pkgLabel := "Disabled"
	if c6.PackageEnabled {
		pkgIcon = display.IconEnabled
		pkgLabel = "Enabled"
	}

	coreIcon := display.IconDisabled
	coreLabel := "Disabled"
	if c6.CoreEnabled {
		coreIcon = display.IconEnabled
		coreLabel = "Enabled"
	}

	fmt.Printf("  %s\n", display.BoldText("C6 State"))
	fmt.Printf("    Package  %s  %s\n", pkgIcon, pkgLabel)
	fmt.Printf("    Core     %s  %s\n", coreIcon, coreLabel)
	fmt.Println()
}

// ── P-State Modification ─────────────────────────────────────────

func modifyPState(idx int, enable, disable bool, fidStr, didStr, vidStr string, freqMHz uint64, vcore float64) {
	oldVal, err := msr.ReadMSR(pstate.PStateMSRs[idx], 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s reading P%d: %v\n",
			display.RedText("Error:"), idx, err)
		os.Exit(1)
	}

	oldPState := pstate.PState(oldVal)
	newPState := oldPState

	// Collect changes
	var changes []string

	if enable {
		newPState = newPState.SetEnable(true)
		changes = append(changes, "Enable")
	}
	if disable {
		newPState = newPState.SetEnable(false)
		changes = append(changes, "Disable")
	}
	// ── Process --freq / --voltage (new high-level API) ──────────
	if freqMHz > 0 {
		// Determine DID: use provided -d or default 8
		did := uint64(pstate.DefaultDID)
		if didStr != "" {
			d, err := pstate.ParseUint(didStr)
			if err == nil {
				did = d
			}
		}
		fid := pstate.FreqToFID(freqMHz, did)
		actualFreq := pstate.FIDToFreq(fid, did)
		newPState = newPState.SetFID(fid)
		newPState = newPState.SetDID(did)
		changes = append(changes, fmt.Sprintf("Freq: %.0fMHz → %.0fMHz", oldPState.FrequencyMHz(), actualFreq))
	}

	if vcore > 0 {
		vid := pstate.VoltageToVID(vcore)
		newPState = newPState.SetVID(vid)
		changes = append(changes, fmt.Sprintf("vCore: %.5fV → %.5fV", oldPState.VCore(), vcore))
	}

	// ── Legacy -f / -d / -v (lower priority) ────────────────────
	if freqMHz == 0 && fidStr != "" {
		fid, err := pstate.ParseUint(fidStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s parsing FID %q: %v\n",
				display.RedText("Error:"), fidStr, err)
			os.Exit(1)
		}
		newPState = newPState.SetFID(fid)
		changes = append(changes, fmt.Sprintf("FID: %d → %d", oldPState.FID(), fid))
	}
	if freqMHz == 0 && didStr != "" {
		did, err := pstate.ParseUint(didStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s parsing DID %q: %v\n",
				display.RedText("Error:"), didStr, err)
			os.Exit(1)
		}
		newPState = newPState.SetDID(did)
		changes = append(changes, fmt.Sprintf("DID: %d → %d", oldPState.DID(), did))
	}
	if vcore == 0 && vidStr != "" {
		vid, err := pstate.ParseUint(vidStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s parsing VID %q: %v\n",
				display.RedText("Error:"), vidStr, err)
			os.Exit(1)
		}
		newPState = newPState.SetVID(vid)
		changes = append(changes, fmt.Sprintf("VID: %d → %d", oldPState.VID(), vid))
	}

	if uint64(newPState) == uint64(oldPState) {
		fmt.Printf("  %s P%d: no changes needed\n", display.YellowText(display.IconBullet), idx)
		return
	}

	// ── Show diff ───────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  %s\n", display.BoldText(fmt.Sprintf("P%d Changes", idx)))
	fmt.Println()

	// Diff table
	dt := display.NewTable("Field", "Before", "After")
	dt.AddRow("Status",
		fmt.Sprintf("%s %s", display.IconEnabled, "Enabled"),
		fmt.Sprintf("%s %s", display.IconEnabled, "Enabled"))
	if !oldPState.Enabled() {
		dt.AddRow("Status",
			display.RedText(display.IconDisabled+" Disabled"),
			display.GreenText(display.IconEnabled+" Enabled"))
	} else if !newPState.Enabled() {
		dt.AddRow("Status",
			display.GreenText(display.IconEnabled+" Enabled"),
			display.RedText(display.IconDisabled+" Disabled"))
	}

	// Show FID/DID row only when using legacy -f/-d
	if freqMHz == 0 && fidStr != "" && newPState.FID() != oldPState.FID() {
		dt.AddRow("FID",
			fmt.Sprintf("%d", oldPState.FID()),
			display.CyanText(fmt.Sprintf("%d", newPState.FID())))
	}
	if freqMHz == 0 && didStr != "" && newPState.DID() != oldPState.DID() {
		dt.AddRow("DID",
			fmt.Sprintf("%d", oldPState.DID()),
			display.CyanText(fmt.Sprintf("%d", newPState.DID())))
	}
	// Show VID row only when using legacy -v
	if vcore == 0 && vidStr != "" && newPState.VID() != oldPState.VID() {
		dt.AddRow("VID",
			fmt.Sprintf("%d", oldPState.VID()),
			display.CyanText(fmt.Sprintf("%d", newPState.VID())))
	}

	// Always show frequency and voltage when state is enabled
	if newPState.Enabled() {
		oldFreq := fmt.Sprintf("%.0f MHz", oldPState.FrequencyMHz())
		newFreq := fmt.Sprintf("%.0f MHz", newPState.FrequencyMHz())
		dt.AddRow("Frequency", oldFreq, display.CyanText(newFreq))

		oldVCore := fmt.Sprintf("%.5f V", oldPState.VCore())
		newVCore := fmt.Sprintf("%.5f V", newPState.VCore())
		dt.AddRow("vCore", oldVCore, display.CyanText(newVCore))
	}

	fmt.Println(dt.Render())

	// ── Lock TSC frequency if needed ────────────────────────────
	tscVal, err := msr.ReadMSR(0xC0010015, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s reading TSC MSR: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}
	if tscVal&(1<<21) == 0 {
		fmt.Printf("  %s\n", display.YellowText("Locking TSC frequency..."))
		numCPUs := msr.CPUCount()
		for c := 0; c < numCPUs; c++ {
			coreTSC, err := msr.ReadMSR(0xC0010015, c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s reading TSC MSR on CPU %d: %v\n",
					display.RedText("Error:"), c, err)
				os.Exit(1)
			}
			if err := msr.WriteMSR(0xC0010015, coreTSC|(1<<21), c); err != nil {
				fmt.Fprintf(os.Stderr, "%s writing TSC MSR on CPU %d: %v\n",
					display.RedText("Error:"), c, err)
				os.Exit(1)
			}
		}
		fmt.Printf("  %s\n", display.GreenText("TSC frequency locked"))
	}

	// ── Write MSR ───────────────────────────────────────────────
	if err := msr.WriteMSR(pstate.PStateMSRs[idx], uint64(newPState), -1); err != nil {
		fmt.Fprintf(os.Stderr, "%s writing P%d: %v\n",
			display.RedText("Error:"), idx, err)
		os.Exit(1)
	}

	fmt.Printf("  %s\n", display.GreenText(fmt.Sprintf("P%d written successfully", idx)))
	fmt.Println()

	// Show final state
	verifyVal, err := msr.ReadMSR(pstate.PStateMSRs[idx], 0)
	if err == nil {
		verifyPState := pstate.PState(verifyVal)
		fmt.Printf("  %s %s\n", display.DimText("Verify →"), verifyPState)
	}
	fmt.Println()
}

// ── C6 Modification ──────────────────────────────────────────────

func modifyC6(enable bool) {
	action := "Enabling"
	verb := "enabled"
	if !enable {
		action = "Disabling"
		verb = "disabled"
	}

	fmt.Println()
	fmt.Printf("  %s C6 State %s...\n", display.BoldText(action), verb)

	// ── Package ────────────────────────────────────────────────
	pkgVal, err := msr.ReadMSR(pstate.C6PackageMSR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s reading C6 package MSR: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}

	oldPkg := pkgVal&pstate.C6PackageMask() != 0
	if enable {
		pkgVal |= pstate.C6PackageMask()
	} else {
		pkgVal &^= pstate.C6PackageMask()
	}
	if err := msr.WriteMSR(pstate.C6PackageMSR, pkgVal, -1); err != nil {
		fmt.Fprintf(os.Stderr, "%s writing C6 package MSR: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}

	// ── Core ───────────────────────────────────────────────────
	coreVal, err := msr.ReadMSR(pstate.C6CoreMSR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s reading C6 core MSR: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}

	oldCore := coreVal&pstate.C6CoreMask() == pstate.C6CoreMask()
	if enable {
		coreVal |= pstate.C6CoreMask()
	} else {
		coreVal &^= pstate.C6CoreMask()
	}
	if err := msr.WriteMSR(pstate.C6CoreMSR, coreVal, -1); err != nil {
		fmt.Fprintf(os.Stderr, "%s writing C6 core MSR: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}

	// ── Result ─────────────────────────────────────────────────
	fmt.Printf("  %s\n", display.GreenText(
		fmt.Sprintf("C6 State %s successfully", verb)))

	// Diff display
	dt := display.NewTable("Domain", "Before", "After")

	oldPkgStr := display.RedText(display.IconDisabled + " Disabled")
	if oldPkg {
		oldPkgStr = display.GreenText(display.IconEnabled + " Enabled")
	}
	newPkgStr := display.RedText(display.IconDisabled + " Disabled")
	if enable {
		newPkgStr = display.GreenText(display.IconEnabled + " Enabled")
	}
	dt.AddRow("Package", oldPkgStr, newPkgStr)

	oldCoreStr := display.RedText(display.IconDisabled + " Disabled")
	if oldCore {
		oldCoreStr = display.GreenText(display.IconEnabled + " Enabled")
	}
	newCoreStr := display.RedText(display.IconDisabled + " Disabled")
	if enable {
		newCoreStr = display.GreenText(display.IconEnabled + " Enabled")
	}
	dt.AddRow("Core", oldCoreStr, newCoreStr)

	fmt.Println()
	fmt.Println(dt.Render())
	fmt.Println()
}

// ── CPU Governor ────────────────────────────────────────────────

// setGovernor sets the CPU frequency scaling governor for all CPUs.
func setGovernor(governor string) {
	data, err := os.ReadFile("/sys/devices/system/cpu/present")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s reading CPU present: %v\n",
			display.RedText("Error:"), err)
		os.Exit(1)
	}

	s := strings.TrimSpace(string(data))
	var maxID int
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		_, err = fmt.Sscanf(s[idx+1:], "%d", &maxID)
	} else {
		_, err = fmt.Sscanf(s, "%d", &maxID)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s parsing CPU present %q: %v\n",
			display.RedText("Error:"), s, err)
		os.Exit(1)
	}

	fmt.Printf("  %s Setting CPU frequency governor to \"%s\" for %d CPUs\n",
		display.Bullet(), governor, maxID+1)

	errors := 0
	for i := 0; i <= maxID; i++ {
		path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_governor", i)
		if err := os.WriteFile(path, []byte(governor+"\n"), 0); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(os.Stderr, "  %s CPU %d: %v\n",
				display.RedText("Error:"), i, err)
			errors++
		}
	}

	if errors > 0 {
		fmt.Printf("  %s %d error(s) occurred\n", display.RedText(display.IconCross), errors)
		os.Exit(1)
	}

	fmt.Printf("  %s\n", display.GreenText(
		fmt.Sprintf("CPU governor set to \"%s\"", governor)))
}
func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  %s\n", display.BoldText("ZenStates-Linux — AMD Ryzen P-State Editor"))
		fmt.Fprintf(os.Stderr, "  %s\n", display.DimText("Dynamically edit AMD Ryzen processor P-States via MSR"))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  %s\n", display.BoldText("Usage: zenstates [options]"))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  %s\n", display.BoldText("Options:"))
		fmt.Fprintf(os.Stderr, "    -l            List all P-States\n")
		fmt.Fprintf(os.Stderr, "    -p <0-7>      P-State to set\n")
		fmt.Fprintf(os.Stderr, "    --enable      Enable P-State\n")
		fmt.Fprintf(os.Stderr, "    --disable     Disable P-State\n")
		fmt.Fprintf(os.Stderr, "    --freq <MHz>  Target frequency (e.g. 3800)\n")
		fmt.Fprintf(os.Stderr, "    --voltage <V>  Target core voltage (e.g. 1.35)\n")
		fmt.Fprintf(os.Stderr, "    -f <val>      FID to set (legacy)\n")
		fmt.Fprintf(os.Stderr, "    -d <val>      DID to set (legacy)\n")
		fmt.Fprintf(os.Stderr, "    -v <val>      VID to set (legacy)\n")
		fmt.Fprintf(os.Stderr, "    --c6-enable   Enable C-State C6\n")
		fmt.Fprintf(os.Stderr, "    --c6-disable  Disable C-State C6\n")
		fmt.Fprintf(os.Stderr, "    --no-color    Disable colored output\n")
		fmt.Fprintf(os.Stderr, "    --governor <g>  Set CPU frequency governor (performance/powersave/...)\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  %s\n", display.BoldText("Examples:"))
		fmt.Fprintf(os.Stderr, "    zenstates -l                          %s\n",
			display.DimText("# List all P-States"))
		fmt.Fprintf(os.Stderr, "    zenstates -p 0 --freq 3800 --voltage 1.35  %s\n",
			display.DimText("# P0: 3800MHz, 1.3500V"))
		fmt.Fprintf(os.Stderr, "    zenstates -p 0 -f 152 -d 8 -v 32       %s\n",
			display.DimText("# Legacy: FID=152 DID=8 VID=32"))
		fmt.Fprintf(os.Stderr, "    zenstates -p 1 --disable             %s\n",
			display.DimText("# Disable P1"))
		fmt.Fprintf(os.Stderr, "    zenstates --governor performance     %s\n",
			display.DimText("# Set CPU governor"))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  %s\n", display.DimText("Requires root and the msr kernel module (modprobe msr)."))
		fmt.Fprintf(os.Stderr, "\n")
	}
}
