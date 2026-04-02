package main

import (
	"context"
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DeviceFingerprint collects hardware identifiers and produces a stable hash.
// The fingerprint survives OS reinstalls (hardware-bound) and catches VM cloning.
//
// Sources (in priority order):
//  1. BIOS Serial Number — motherboard-level, survives OS reinstall
//  2. CPU Processor ID — silicon-level, unchangeable
//  3. Boot disk serial — drive-level, survives OS reinstall
//  4. Windows Machine GUID — OS-level, catches VM clones
//
// If any source is unavailable, fallbacks are used so the fingerprint
// always has at least 2 components. The result is a SHA-256 hash.

func collectFingerprint() string {
	components := []string{}
	sources := 0

	// 1. BIOS Serial
	if v := wmicGet("bios", "SerialNumber"); v != "" {
		components = append(components, "BIOS:"+v)
		sources++
	}

	// 2. CPU Processor ID
	if v := wmicGet("cpu", "ProcessorId"); v != "" {
		components = append(components, "CPU:"+v)
		sources++
	}

	// 3. Boot disk serial
	if v := wmicGet("diskdrive where Index=0", "SerialNumber"); v != "" {
		components = append(components, "DISK:"+v)
		sources++
	}

	// 4. Windows Machine GUID (registry)
	if v := readMachineGUID(); v != "" {
		components = append(components, "MGUID:"+v)
		sources++
	}

	// Fallbacks if we got fewer than 2 primary sources
	if sources < 2 {
		if v := wmicGet("baseboard", "SerialNumber"); v != "" {
			components = append(components, "MB:"+v)
			sources++
		}
	}
	if sources < 2 {
		if v := wmicGet("os", "SerialNumber"); v != "" {
			components = append(components, "WINID:"+v)
			sources++
		}
	}
	if sources < 2 {
		if v := wmicGet("csproduct", "UUID"); v != "" && v != "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF" {
			components = append(components, "UUID:"+v)
			sources++
		}
	}

	// Last resort — volume serial of C: drive
	if len(components) == 0 {
		if v := volumeSerial(); v != "" {
			components = append(components, "VOL:"+v)
		}
	}

	// If we still have fewer than 2 components, the fingerprint is weak.
	// Use a persistent random ID stored on disk as fallback.
	if len(components) < 2 {
		if persistID := loadOrCreatePersistentID(); persistID != "" {
			components = append(components, "PERSIST:"+persistID)
		}
	}

	raw := strings.Join(components, "|")
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// wmicGet runs: wmic <class> get <field> /value — returns trimmed value or "".
func wmicGet(class, field string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{}
	// class might contain "where" clause like "diskdrive where Index=0"
	args = append(args, strings.Fields(class)...)
	args = append(args, "get", field, "/value")

	cmd := exec.CommandContext(ctx, "wmic", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" && !isGenericID(val) {
					return val
				}
			}
		}
	}
	return ""
}

// readMachineGUID reads Windows Machine GUID from registry via reg.exe.
// Works without admin privileges.
func readMachineGUID() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "reg", "query",
		`HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "MachineGuid") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// volumeSerial gets C: volume serial as a last-resort fallback.
func volumeSerial() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cmd", "/c", "vol", "C:")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Serial Number") {
			parts := strings.SplitN(line, "is", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// isGenericID returns true if a hardware ID is a placeholder value
// that manufacturers use instead of a real serial.
func isGenericID(val string) bool {
	lower := strings.ToLower(strings.TrimSpace(val))
	generics := []string{
		"to be filled by o.e.m.",
		"to be filled",
		"default string",
		"not available",
		"none",
		"n/a",
		"system serial number",
		"system product name",
		"0",
		"123456789",
	}
	for _, g := range generics {
		if lower == g {
			return true
		}
	}
	return false
}

// loadOrCreatePersistentID returns a stable random ID persisted to disk.
// Used as a fallback when fewer than 2 hardware sources are available.
func loadOrCreatePersistentID() string {
	dir, _ := os.UserConfigDir()
	path := filepath.Join(dir, "Dotachi", ".device_id")

	// Try to read existing
	data, err := os.ReadFile(path)
	if err == nil && len(data) >= 32 {
		return strings.TrimSpace(string(data))
	}

	// Generate new
	b := make([]byte, 16)
	if _, err := crypto_rand.Read(b); err != nil {
		return ""
	}
	id := hex.EncodeToString(b)

	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(id), 0o600)
	return id
}

// GetDeviceFingerprint is exposed to the Wails frontend.
// Returns the SHA-256 hash of this device's hardware identifiers.
func (a *App) GetDeviceFingerprint() string {
	return collectFingerprint()
}
