// Package msr provides read/write access to CPU Model-Specific Registers (MSR)
// via the /dev/cpu/N/msr device files.
package msr

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const msrDevice = "/dev/cpu/%d/msr"

// cpuCount returns the number of MSR device files found.
func cpuCount() int {
	matches, err := filepath.Glob("/dev/cpu/[0-9]*/msr")
	if err != nil {
		return 0
	}
	return len(matches)
}

// WriteMSR writes an 8-byte value to the specified MSR register.
// If cpu is -1, writes to all CPUs.
func WriteMSR(msr uint32, val uint64, cpu int) error {
	if cpu == -1 {
		matches, err := filepath.Glob("/dev/cpu/[0-9]*/msr")
		if err != nil {
			return fmt.Errorf("glob failed: %w", err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("msr module not loaded (run modprobe msr)")
		}
		for _, m := range matches {
			if err := writeMSRFile(m, msr, val); err != nil {
				return err
			}
		}
		return nil
	}

	path := fmt.Sprintf(msrDevice, cpu)
	return writeMSRFile(path, msr, val)
}

func writeMSRFile(path string, msr uint32, val uint64) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("msr module not loaded (run modprobe msr): %w", err)
	}
	defer f.Close()

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, val)
	if _, err := f.WriteAt(buf, int64(msr)); err != nil {
		return fmt.Errorf("write msr 0x%X to %s: %w", msr, path, err)
	}
	return nil
}

// ReadMSR reads an 8-byte value from the specified MSR register on the given CPU.
func ReadMSR(msr uint32, cpu int) (uint64, error) {
	path := fmt.Sprintf(msrDevice, cpu)
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return 0, fmt.Errorf("msr module not loaded (run modprobe msr): %w", err)
	}
	defer f.Close()

	buf := make([]byte, 8)
	if _, err := f.ReadAt(buf, int64(msr)); err != nil {
		return 0, fmt.Errorf("read msr 0x%X from %s: %w", msr, path, err)
	}

	return binary.LittleEndian.Uint64(buf), nil
}

// CPUCount returns the number of MSR-capable CPU cores.
func CPUCount() int {
	return cpuCount()
}

// ParseCPUList parses a comma-separated list of CPU numbers or ranges.
// Example: "0,2-4" -> [0,2,3,4]. Returns -1 for all CPUs.
func ParseCPUList(s string) ([]int, error) {
	if s == "" || s == "all" {
		return nil, nil // nil means all CPUs
	}

	var cpus []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			parts := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid CPU range %q: %w", part, err)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid CPU range %q: %w", part, err)
			}
			for i := start; i <= end; i++ {
				cpus = append(cpus, i)
			}
		} else {
			cpu, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid CPU number %q: %w", part, err)
			}
			cpus = append(cpus, cpu)
		}
	}
	return cpus, nil
}
