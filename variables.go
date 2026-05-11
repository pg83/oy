package main

import (
	"strings"
)

// VariableSet provides an interface for querying and resolving variable values
// during conditional evaluation. Implementations must support truthiness testing
// (IsTrue) for boolean conditions in IF/ELSEIF/BUILD_ONLY_IF statements.
type VariableSet interface {
	SetValue(key, value string)
	GetValue(key string) (string, bool)
	IsTrue(key string) bool
	HasKey(key string) bool
	GetArchValue() string
	GetMuslValue() bool
	GetOSValue() string
}

// MemoryVariableSet provides an in-memory implementation of VariableSet backed
// by a map[string]string. Truthiness follows ya.make semantics: "yes", "true",
// "on", "1" evaluate to true; "no", "false", "off", "0", and empty strings
// evaluate to false. All other non-empty strings evaluate to true.
type MemoryVariableSet struct {
	values map[string]string
}

// NewMemoryVariableSet creates an empty MemoryVariableSet.
func NewMemoryVariableSet() *MemoryVariableSet {
	return &MemoryVariableSet{
		values: make(map[string]string),
	}
}

// SetValue stores a key-value pair in the variable set.
func (m *MemoryVariableSet) SetValue(key, value string) {
	m.values[key] = value
}

// GetValue retrieves the value for the given key, returning (value, true) if found,
// ("", false) otherwise.
func (m *MemoryVariableSet) GetValue(key string) (string, bool) {
	val, ok := m.values[key]
	return val, ok
}

// IsTrue returns true if the key resolves to a true value according to ya.make
// semantics. Positive values: "yes", "true", "on", "1". False values: "no",
// "false", "off", "0", or empty/unset keys. All other non-empty strings return true.
func (m *MemoryVariableSet) IsTrue(key string) bool {
	val, ok := m.values[key]
	if !ok {
		return false
	}

	trimmed := strings.ToLower(strings.TrimSpace(val))

	if trimmed == "" {
		return false
	}

	if trimmed == "yes" || trimmed == "true" || trimmed == "on" || trimmed == "1" {
		return true
	}

	if trimmed == "no" || trimmed == "false" || trimmed == "off" || trimmed == "0" {
		return false
	}

	return true
}

// HasKey returns true if the key exists in the variable set.
func (m *MemoryVariableSet) HasKey(key string) bool {
	_, ok := m.values[key]
	return ok
}

func (m *MemoryVariableSet) GetArchValue() string {
	return ""
}

func (m *MemoryVariableSet) GetMuslValue() bool {
	return false
}

func (m *MemoryVariableSet) GetOSValue() string {
	return ""
}

// BuildContextVariableSet wraps a BuildContext and implements VariableSet, resolving
// built-in platform constants (OS_LINUX, OS_WINDOWS, OS_DARWIN, OS_MAC), architecture
// constants (ARCH_TYPE_64, ARCH_TYPE_32), and compiler constants (MSVC, CLANG, GCC).
// Unknown keys are delegated to an inner VariableSet for user-defined flags.
type BuildContextVariableSet struct {
	ctx   *BuildContext
	inner VariableSet
}

// NewBuildContextVariableSet creates a BuildContextVariableSet that resolves platform
// constants from the BuildContext and delegates other queries to the inner set.
func NewBuildContextVariableSet(ctx *BuildContext, inner VariableSet) *BuildContextVariableSet {
	return &BuildContextVariableSet{
		ctx:   ctx,
		inner: inner,
	}
}

// SetValue delegates to the inner VariableSet.
func (b *BuildContextVariableSet) SetValue(key, value string) {
	b.inner.SetValue(key, value)
}

// GetValue delegates to the inner VariableSet.
func (b *BuildContextVariableSet) GetValue(key string) (string, bool) {
	return b.inner.GetValue(key)
}

// IsTrue resolves built-in constants from BuildContext and delegates unknown keys
// to the inner set. OS constants: OS_LINUX, OS_WINDOWS, OS_DARWIN, OS_MAC (legacy).
// Architecture: ARCH_TYPE_64, ARCH_TYPE_32, ARCH_X86_64, ARCH_AARCH64. Compiler: MSVC, CLANG, GCC.
// Platform flags: MUSL, TARGET_PLATFORM_*, language flags: PROTO, GO, PYTHON3, etc.
func (b *BuildContextVariableSet) IsTrue(key string) bool {
	switch key {
	case "OS_LINUX":
		return b.ctx.Platform == PlatformLinux
	case "OS_WINDOWS":
		return b.ctx.Platform == PlatformWindows
	case "OS_DARWIN", "OS_MAC":
		return b.ctx.Platform == PlatformDarwin
	case "ARCH_TYPE_64":
		return b.ctx.Arch == Arch64
	case "ARCH_TYPE_32":
		return b.ctx.Arch == Arch32
	case "ARCH_X86_64":
		return b.ctx.ArchString == "x86_64"
	case "ARCH_AARCH64":
		return b.ctx.ArchString == "aarch64"
	case "ARCH_ARM64":
		return b.ctx.ArchString == "arm64" || b.ctx.ArchString == "aarch64"
	case "ARCH_ARM6":
		return b.ctx.ArchString == "armv6"
	case "ARCH_ARM7":
		return b.ctx.ArchString == "armv7"
	case "ARCH_ARM":
		return strings.HasPrefix(b.ctx.ArchString, "arm") || b.ctx.ArchString == "aarch64"
	case "ARCH_PPC64LE":
		return b.ctx.ArchString == "ppc64le"
	case "ARCH_RISCV32":
		return b.ctx.ArchString == "riscv32"
	case "ARCH_RISCV64":
		return b.ctx.ArchString == "riscv64"
	case "MSVC":
		return b.ctx.Compiler == CompilerMSVC
	case "CLANG":
		return b.ctx.Compiler == CompilerClang
	case "GCC":
		return b.ctx.Compiler == CompilerGCC
	case "MUSL":
		return b.ctx.Musl
	default:
		if b.isTargetPlatformKey(key) {
			return b.resolveTargetPlatform(key)
		}
		if b.isLanguageKey(key) {
			return b.resolveLanguage(key)
		}
		return b.inner.IsTrue(key)
	}
}

// HasKey returns true for all built-in constants and delegates unknown keys to the inner set.
func (b *BuildContextVariableSet) HasKey(key string) bool {
	switch key {
	case "OS_LINUX", "OS_WINDOWS", "OS_DARWIN", "OS_MAC":
		return true
	case "ARCH_TYPE_64", "ARCH_TYPE_32", "ARCH_X86_64", "ARCH_AARCH64":
		return true
	case "ARCH_ARM64", "ARCH_ARM6", "ARCH_ARM7", "ARCH_ARM", "ARCH_PPC64LE", "ARCH_RISCV32", "ARCH_RISCV64":
		return true
	case "MSVC", "CLANG", "GCC":
		return true
	case "MUSL":
		return true
	default:
		if b.isTargetPlatformKey(key) || b.isLanguageKey(key) {
			return true
		}
		return b.inner.HasKey(key)
	}
}

func (b *BuildContextVariableSet) isTargetPlatformKey(key string) bool {
	if len(key) <= 16 {
		return false
	}
	return key[:16] == "TARGET_PLATFORM_"
}

func (b *BuildContextVariableSet) resolveTargetPlatform(key string) bool {
	if b.ctx.TargetPlatform == "" {
		return false
	}

	normalizedTarget := strings.ToUpper(strings.ReplaceAll(b.ctx.TargetPlatform, "-", "_"))
	expected := "TARGET_PLATFORM_" + normalizedTarget
	return key == expected
}

func (b *BuildContextVariableSet) isLanguageKey(key string) bool {
	languageKeys := []string{"PROTO", "GO", "PYTHON3", "PYTHON", "JAVA", "CSHARP", "CPP"}
	for _, lang := range languageKeys {
		if key == lang {
			return true
		}
	}
	return false
}

func (b *BuildContextVariableSet) resolveLanguage(key string) bool {
	if b.ctx.Language == "" {
		return false
	}
	return strings.ToUpper(b.ctx.Language) == key
}

func (b *BuildContextVariableSet) GetArchValue() string {
	return b.ctx.ArchString
}

func (b *BuildContextVariableSet) GetMuslValue() bool {
	return b.ctx.Musl
}

func (b *BuildContextVariableSet) GetOSValue() string {
	switch b.ctx.Platform {
	case PlatformLinux:
		return "linux"
	case PlatformWindows:
		return "windows"
	case PlatformDarwin:
		return "darwin"
	default:
		return ""
	}
}
