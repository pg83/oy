package main

import (
	"runtime"
)

// BuildContext represents build configuration including platform, architecture,
// compiler, and language-specific flags that affect conditional evaluation.
type BuildContext struct {
	Platform       Platform
	Arch           Arch
	Compiler       Compiler
	Flags          map[string]bool
	Languages      map[string]bool
	TargetPlatform string
	Musl           bool
	PlatformFlags  map[string]string
	Language       string
}

// Platform represents the target operating system for conditional evaluation.
type Platform int

const (
	PlatformLinux Platform = iota
	PlatformWindows
	PlatformDarwin
)

// Arch represents the target machine architecture.
type Arch int

const (
	Arch32 Arch = iota
	Arch64
)

// Compiler represents the target compiler toolchain.
type Compiler int

const (
	CompilerGCC Compiler = iota
	CompilerClang
	CompilerMSVC
)

// NewBuildContext creates a BuildContext with defaults detected from the current
// runtime environment. Platform and architecture are detected via runtime.GOOS
// and runtime.GOARCH. Compiler defaults to GCC on Linux, Clang on macOS/Darwin,
// and MSVC on Windows.
func NewBuildContext() *BuildContext {
	ctx := &BuildContext{
		Flags:         make(map[string]bool),
		Languages:     make(map[string]bool),
		PlatformFlags: make(map[string]string),
	}

	switch runtime.GOOS {
	case "linux":
		ctx.Platform = PlatformLinux
	case "windows":
		ctx.Platform = PlatformWindows
	case "darwin":
		ctx.Platform = PlatformDarwin
	default:
		ctx.Platform = PlatformLinux
	}

	switch runtime.GOARCH {
	case "386":
		ctx.Arch = Arch32
	case "amd64", "arm64":
		ctx.Arch = Arch64
	default:
		ctx.Arch = Arch64
	}

	if ctx.Platform == PlatformDarwin {
		ctx.Compiler = CompilerClang
	} else if ctx.Platform == PlatformWindows {
		ctx.Compiler = CompilerMSVC
	} else {
		ctx.Compiler = CompilerGCC
	}

	return ctx
}
