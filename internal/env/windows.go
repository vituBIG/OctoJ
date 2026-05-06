//go:build windows

package env

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/windows/registry"
)

type windowsManager struct {
	octojHome string
}

// NewManager returns the Windows environment manager.
func NewManager(octojHome string) (Manager, error) {
	return &windowsManager{octojHome: octojHome}, nil
}

func newWindowsManager(octojHome string) Manager {
	return &windowsManager{octojHome: octojHome}
}

func (m *windowsManager) Plan() ([]string, error) {
	var changes []string

	current, err := m.readUserEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to read user environment: %w", err)
	}

	octojHomeVal := m.octojHome
	javaHomeVal := `%OCTOJ_HOME%\current`
	pathAdditions := []string{`%OCTOJ_HOME%\bin`, `%JAVA_HOME%\bin`}

	if current["OCTOJ_HOME"] != octojHomeVal {
		changes = append(changes, fmt.Sprintf("SET OCTOJ_HOME=%s", octojHomeVal))
	}

	if current["JAVA_HOME"] != javaHomeVal {
		changes = append(changes, fmt.Sprintf("SET JAVA_HOME=%s", javaHomeVal))
	}

	currentPath := current["PATH"]
	alreadyFirst := strings.HasPrefix(strings.ToLower(currentPath), strings.ToLower(pathAdditions[0]+";"))
	if !alreadyFirst {
		changes = append(changes, fmt.Sprintf("PREPEND to PATH (front): %s", strings.Join(pathAdditions, ";")))
	}

	return changes, nil
}

func (m *windowsManager) Apply() error {
	// Set OCTOJ_HOME
	if err := setxEnv("OCTOJ_HOME", m.octojHome); err != nil {
		return fmt.Errorf("failed to set OCTOJ_HOME: %w", err)
	}

	// Set JAVA_HOME to expand from OCTOJ_HOME
	if err := setRegistryExpandString("JAVA_HOME", `%OCTOJ_HOME%\current`); err != nil {
		return fmt.Errorf("failed to set JAVA_HOME: %w", err)
	}

	// Update PATH
	if err := m.updatePath(); err != nil {
		return fmt.Errorf("failed to update PATH: %w", err)
	}

	// Broadcast WM_SETTINGCHANGE so Explorer and new terminals pick up the changes
	broadcastEnvChange()

	return nil
}

func (m *windowsManager) PrintRestartInstructions() {
	fmt.Println("Environment variables have been set in the Windows user registry.")
	fmt.Println("Please restart your terminal (or log out and log back in) for the changes to take effect.")
	fmt.Println()
	fmt.Println("To verify, open a new terminal and run:")
	fmt.Println("  echo %OCTOJ_HOME%")
	fmt.Println("  echo %JAVA_HOME%")
	fmt.Println("  java -version")
}

// readUserEnv reads the current user environment variables from the registry.
func (m *windowsManager) readUserEnv() (map[string]string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.READ)
	if err != nil {
		return map[string]string{}, nil
	}
	defer k.Close()

	names, err := k.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}

	env := make(map[string]string)
	for _, name := range names {
		val, _, err := k.GetStringValue(name)
		if err == nil {
			env[strings.ToUpper(name)] = val
		}
	}

	return env, nil
}

// setxEnv uses SetX to set a user environment variable (persisted to registry).
func setxEnv(name, value string) error {
	cmd := exec.Command("setx", name, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("setx %s failed: %w\nOutput: %s", name, err, string(output))
	}
	return nil
}

// setRegistryExpandString sets an expandable string value in HKCU\Environment.
func setRegistryExpandString(name, value string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()

	return k.SetExpandStringValue(name, value)
}

// updatePath ensures OctoJ entries are at the FRONT of the user PATH in the registry.
// It removes any existing OctoJ entries first (wherever they are) then prepends them,
// so they always beat system-level Java installations.
func (m *windowsManager) updatePath() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()

	currentPath, _, err := k.GetStringValue("PATH")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to read PATH: %w", err)
	}

	octojEntries := []string{`%OCTOJ_HOME%\bin`, `%JAVA_HOME%\bin`}

	// Strip any existing OctoJ entries from wherever they are in PATH
	parts := strings.Split(currentPath, ";")
	var kept []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		isOctoJ := false
		for _, e := range octojEntries {
			if strings.EqualFold(trimmed, e) {
				isOctoJ = true
				break
			}
		}
		if !isOctoJ {
			kept = append(kept, trimmed)
		}
	}

	// Prepend OctoJ entries so they come first in User PATH
	newPath := strings.Join(octojEntries, ";") + ";" + strings.Join(kept, ";")
	newPath = strings.TrimSuffix(newPath, ";")

	return k.SetExpandStringValue("PATH", newPath)
}

// containsPathEntry checks if a path entry exists in a semicolon-separated path string.
func containsPathEntry(pathStr, entry string) bool {
	pathStr = strings.ToLower(pathStr)
	entry = strings.ToLower(entry)
	for _, part := range strings.Split(pathStr, ";") {
		if strings.TrimSpace(part) == entry {
			return true
		}
	}
	return false
}

// broadcastEnvChange notifies other processes that environment variables have changed.
func broadcastEnvChange() {
	// Use PowerShell to broadcast WM_SETTINGCHANGE
	script := `
[System.Environment]::GetEnvironmentVariable('PATH', 'User') | Out-Null
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class WinEnv {
    [DllImport("user32.dll", SetLastError=true, CharSet=CharSet.Auto)]
    public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
}
"@
$HWND_BROADCAST = [IntPtr]0xffff
$WM_SETTINGCHANGE = 0x001A
$result = [UIntPtr]::Zero
[WinEnv]::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result)
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	_ = cmd.Run() // best-effort, don't fail if this doesn't work
}

// PlanRemoval returns a description of the registry changes Remove() will make.
func (m *windowsManager) PlanRemoval() ([]string, error) {
	current, err := m.readUserEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to read user environment: %w", err)
	}

	var changes []string

	if _, ok := current["OCTOJ_HOME"]; ok {
		changes = append(changes, "DELETE OCTOJ_HOME from user environment")
	}

	// Only remove JAVA_HOME if it still points to our value
	if jh, ok := current["JAVA_HOME"]; ok && strings.Contains(strings.ToLower(jh), "octoj") {
		changes = append(changes, "DELETE JAVA_HOME from user environment")
	}

	pathEntries := []string{`%OCTOJ_HOME%\bin`, `%JAVA_HOME%\bin`}
	currentPath := current["PATH"]
	for _, e := range pathEntries {
		if containsPathEntry(currentPath, e) {
			changes = append(changes, fmt.Sprintf("REMOVE from PATH: %s", e))
		}
	}

	if len(changes) == 0 {
		changes = append(changes, "No OctoJ environment variables found — nothing to remove")
	}

	return changes, nil
}

// Remove deletes OCTOJ_HOME, JAVA_HOME (if ours) and removes OctoJ entries from PATH.
func (m *windowsManager) Remove() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()

	// Delete OCTOJ_HOME
	if err := k.DeleteValue("OCTOJ_HOME"); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to delete OCTOJ_HOME: %w", err)
	}

	// Delete JAVA_HOME only if it still references octoj
	javaHome, _, _ := k.GetStringValue("JAVA_HOME")
	if strings.Contains(strings.ToLower(javaHome), "octoj") {
		if err := k.DeleteValue("JAVA_HOME"); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("failed to delete JAVA_HOME: %w", err)
		}
	}

	// Remove OctoJ entries from PATH
	currentPath, _, err := k.GetStringValue("PATH")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to read PATH: %w", err)
	}

	octojEntries := []string{`%OCTOJ_HOME%\bin`, `%JAVA_HOME%\bin`}
	parts := strings.Split(currentPath, ";")
	var kept []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		isOctoJ := false
		for _, e := range octojEntries {
			if strings.EqualFold(trimmed, e) {
				isOctoJ = true
				break
			}
		}
		if trimmed != "" && !isOctoJ {
			kept = append(kept, trimmed)
		}
	}

	newPath := strings.Join(kept, ";")
	if newPath != currentPath {
		if err := k.SetExpandStringValue("PATH", newPath); err != nil {
			return fmt.Errorf("failed to update PATH: %w", err)
		}
	}

	broadcastEnvChange()
	return nil
}

// readSystemPath reads the machine-level PATH from the registry.
func readSystemPath() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
		registry.READ)
	if err != nil {
		return "", err
	}
	defer k.Close()
	val, _, err := k.GetStringValue("PATH")
	return val, err
}

// IsJavaInSystemPath reports whether javaExe lives inside any directory listed in the System PATH.
func IsJavaInSystemPath(javaExe string) bool {
	systemPath, err := readSystemPath()
	if err != nil {
		return false
	}
	javaLower := filepath.ToSlash(strings.ToLower(javaExe))
	for _, entry := range strings.Split(systemPath, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		entryLower := strings.TrimSuffix(filepath.ToSlash(strings.ToLower(entry)), "/")
		if strings.HasPrefix(javaLower, entryLower+"/") {
			return true
		}
	}
	return false
}

// PrependToSystemPath adds OctoJ's bin directories at the front of the Windows System PATH.
// It launches an elevated PowerShell process, which triggers a UAC prompt.
func PrependToSystemPath(octojHome string) error {
	binDir := octojHome + `\bin`
	currentBinDir := octojHome + `\current\bin`

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$regPath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
$currentPath = (Get-ItemProperty -Path $regPath -Name PATH).PATH
$entries = $currentPath -split ';' | Where-Object { $_ -ne '' }
$octojEntries = @('%s', '%s')
$kept = $entries | Where-Object { $e = $_; -not ($octojEntries | Where-Object { $_ -ieq $e }) }
$newPath = ($octojEntries + $kept) -join ';'
Set-ItemProperty -Path $regPath -Name PATH -Value $newPath
`, binDir, currentBinDir)

	encoded := encodePSCommand(script)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf(`Start-Process powershell.exe -ArgumentList '-NoProfile -NonInteractive -EncodedCommand %s' -Verb RunAs -Wait`, encoded))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(output)))
	}
	broadcastEnvChange()
	return nil
}

// encodePSCommand encodes a PowerShell script as UTF-16 LE base64 for use with -EncodedCommand.
func encodePSCommand(script string) string {
	u16 := utf16.Encode([]rune(script))
	b := make([]byte, len(u16)*2)
	for i, c := range u16 {
		b[i*2] = byte(c)
		b[i*2+1] = byte(c >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// isWindowsBuild is used by env.go to select the right implementation.
var _ Manager = (*windowsManager)(nil)

// Ensure this file is only compiled on Windows
var _ = os.PathSeparator
