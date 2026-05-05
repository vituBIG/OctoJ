//go:build !windows

package installer

func EnsureShims(_ string) error { return nil }
