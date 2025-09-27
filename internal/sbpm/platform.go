package sbpm

import (
	"runtime"
)

// Platform captures minimal OS/arch data. OS values: windows, linux, macos
// Arch mirrors Go runtime.GOARCH.
//
type Platform struct {
	OS   string
	Arch string
}

// DetectPlatform returns the current platform, allowing an optional override
// for OS name (windows|linux|macos). Any other override is ignored.
func DetectPlatform(override string) Platform {
	osName := runtime.GOOS
	switch override {
	case "windows", "linux", "macos":
		osName = map[string]string{"macos": "darwin", "windows": "windows", "linux": "linux"}[override]
	}
	if osName == "darwin" {
		return Platform{OS: "macos", Arch: runtime.GOARCH}
	}
	return Platform{OS: osName, Arch: runtime.GOARCH}
}
