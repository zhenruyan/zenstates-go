// Command togglecode turns on/off the Q-Code display on ASUS Crosshair VI Hero
// motherboards (and other boards with a compatible Super I/O chip).
//
// It uses /dev/port to perform port I/O on the LPC Super I/O controller
// registers at ports 0x2E/0x2F.
//
// Requires root access (or CAP_SYS_RAWIO capability).
package main

import (
	"fmt"
	"os"

	"github.com/amdcpu-overclocking/internal/display"
)

const (
	superIOPort = 0x2E
	superIOData = 0x2F
)

// outb writes a byte to the specified I/O port via /dev/port.
func outb(port uint16, val byte, dev *os.File) error {
	_, err := dev.Seek(int64(port), 0)
	if err != nil {
		return fmt.Errorf("seek to port 0x%X: %w", port, err)
	}
	_, err = dev.Write([]byte{val})
	if err != nil {
		return fmt.Errorf("write to port 0x%X: %w", port, err)
	}
	return nil
}

// inb reads a byte from the specified I/O port via /dev/port.
func inb(port uint16, dev *os.File) (byte, error) {
	_, err := dev.Seek(int64(port), 0)
	if err != nil {
		return 0, fmt.Errorf("seek to port 0x%X: %w", port, err)
	}
	buf := make([]byte, 1)
	_, err = dev.Read(buf)
	if err != nil {
		return 0, fmt.Errorf("read from port 0x%X: %w", port, err)
	}
	return buf[0], nil
}

func main() {
	fmt.Println()
	fmt.Printf("  %s\n", display.BoldText("ASUS Q-Code Display Toggle"))
	fmt.Printf("  %s\n\n", display.DimText("Super I/O: 0x2E/0x2F · GPIO LD7 · Reg 0xF0"))

	// Open /dev/port for I/O port access
	dev, err := os.OpenFile("/dev/port", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s %v\n", display.RedText("Error:"), err)
		fmt.Fprintf(os.Stderr, "  %s\n", display.YellowText("Run as root or grant CAP_SYS_RAWIO capability"))
		os.Exit(1)
	}
	defer dev.Close()

	// ── Enter Super I/O conﬁg mode ──────────────────────────────
	fmt.Printf("  %s Entering Super I/O config mode...\n", display.Bullet())
	if err := outb(superIOPort, 0x87, dev); err != nil {
		fatal("entry sequence", err)
	}
	if err := outb(superIOPort, 0x01, dev); err != nil {
		fatal("entry sequence", err)
	}
	if err := outb(superIOPort, 0x55, dev); err != nil {
		fatal("entry sequence", err)
	}
	if err := outb(superIOPort, 0x55, dev); err != nil {
		fatal("entry sequence", err)
	}

	// ── Select logical device 7 (GPIO) ───────────────────────────
	fmt.Printf("  %s Selecting GPIO logical device (LD7)...\n", display.Bullet())
	if err := outb(superIOPort, 0x07, dev); err != nil {
		fatal("select device", err)
	}
	if err := outb(superIOData, 0x03, dev); err != nil {
		fatal("set device number", err)
	}

	// ── Read GPIO register 0xF0 ─────────────────────────────────
	fmt.Printf("  %s Reading GPIO register 0xF0...\n", display.Bullet())
	if err := outb(superIOPort, 0xF0, dev); err != nil {
		fatal("select GPIO register", err)
	}

	val, err := inb(superIOData, dev)
	if err != nil {
		fatal("read GPIO register", err)
	}

	oldState := val & 0x08
	oldLabel := "ON"
	if oldState == 0 {
		oldLabel = "OFF"
	}

	fmt.Printf("  %s Current state: Q-Code = %s (0xF0 = 0x%02X, bit3=%d)\n",
		display.Bullet(),
		display.BoldText(oldLabel),
		val,
		(oldState>>3)&1)

	// ── Toggle bit 3 ────────────────────────────────────────────
	val ^= 0x08
	newState := val & 0x08
	newLabel := "ON"
	if newState == 0 {
		newLabel = "OFF"
	}

	fmt.Printf("  %s Toggling bit 3...\n", display.Bullet())

	if err := outb(superIOData, val, dev); err != nil {
		fatal("write GPIO register", err)
	}

	// ── Result ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  %s\n", display.GreenText(display.IconCheck+" Q-Code display toggled successfully!"))
	fmt.Println()
	fmt.Printf("  %s Q-Code: %s → %s\n",
		display.Bullet(),
		display.RedText(oldLabel),
		display.GreenText(newLabel))
	fmt.Printf("  %s Reg 0xF0: 0x%02X → 0x%02X  (bit3: %d→%d)\n",
		display.Bullet(),
		val^0x08, val,
		(oldState>>3)&1, (newState>>3)&1)
	fmt.Println()
}

func fatal(context string, err error) {
	fmt.Fprintf(os.Stderr, "  %s %s: %v\n", display.RedText("Error:"), context, err)
	os.Exit(1)
}
