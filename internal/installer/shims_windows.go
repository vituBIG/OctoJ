//go:build windows

package installer

import (
	"os"
	"path/filepath"
)

// javaShims lists the Java tools to shim in %OCTOJ_HOME%\bin.
var javaShims = []string{
	"java", "javac", "javaw", "jar", "javadoc",
	"keytool", "jshell", "javap", "jlink", "jpackage",
}

// shimScript is a .cmd that delegates to the same-named exe under current\bin.
// %~dp0 expands to the directory containing the .cmd file.
// %~n0  expands to the script filename without extension (e.g. "java").
const shimScript = "@echo off\r\n\"%~dp0\\..\\current\\bin\\%~n0.exe\" %*\r\n"

// EnsureShims creates .cmd wrapper scripts in binDir for common Java tools.
// They forward to %OCTOJ_HOME%\current\bin\<tool>.exe so that switching the
// current junction immediately affects any terminal where %OCTOJ_HOME%\bin
// appears first in PATH.
func EnsureShims(binDir string) error {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	for _, name := range javaShims {
		p := filepath.Join(binDir, name+".cmd")
		if _, err := os.Stat(p); err == nil {
			continue // already exists
		}
		if err := os.WriteFile(p, []byte(shimScript), 0o644); err != nil {
			return err
		}
	}
	return nil
}
