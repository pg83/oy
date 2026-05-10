package main

import (
	"fmt"
	"strings"
)

type BuildConfig struct {
	AllocatorType   string
	OS              string
	Arch            string
	Compiler        string
	Musl            bool
	Flags           map[string]string
	ValidAllocators map[string]bool
}

func NewBuildConfig(ctx *ParseContext) *BuildConfig {
	config := &BuildConfig{
		Musl:            ctx.Musl,
		Flags:           make(map[string]string),
		ValidAllocators: make(map[string]bool),
	}

	validTypes := []string{
		"LF", "LF_YT", "LF_DBG", "YT", "YT_TCMALLOC", "YT_TCMALLOC_256K", "J", "B", "BM", "C",
		"TCMALLOC", "TCMALLOC_SMALL_BUT_SLOW", "TCMALLOC_NUMA_256K",
		"TCMALLOC_NUMA_LARGE_PAGES", "TCMALLOC_256K", "TCMALLOC_TC",
		"GOOGLE", "LOCKLESS", "SYSTEM", "FAKE", "MIM", "MIM_SDC",
		"HU", "PROFILED_HU", "THREAD_PROFILED_HU",
	}

	for _, t := range validTypes {
		config.ValidAllocators[t] = true
	}

	config.OS = detectOS(ctx)
	config.Arch = detectArch(ctx)
	config.Compiler = detectCompiler(ctx)

	config.AllocatorType = config.determineDefaultAllocator()

	return config
}

func (bc *BuildConfig) determineDefaultAllocator() string {
	if bc.Musl {
		return "TCMALLOC_TC"
	}

	if bc.OS == "windows" || bc.OS == "android" || bc.Arch == "32bit" {
		return "J"
	}

	if bc.OS == "linux" && bc.Compiler != "gcc" && bc.Arch == "x86_64" {
		return "TCMALLOC_TC"
	}

	if bc.Arch == "xtensa" {
		return "FAKE"
	}

	return "SYSTEM"
}

func (bc *BuildConfig) resolveAllocatorPEERDIRs(allocatorType string) []string {
	allocType := allocatorType

	if !bc.ValidAllocators[allocType] {
		return []string{}
	}

	mappings := map[string][]string{
		"MIM":                       {"library/cpp/malloc/mimalloc"},
		"MIM_SDC":                   {"library/cpp/malloc/mimalloc_sdc"},
		"HU":                        {"library/cpp/malloc/hu"},
		"PROFILED_HU":               {"library/cpp/malloc/profiled_hu"},
		"THREAD_PROFILED_HU":        {"library/cpp/malloc/thread_profiled_hu"},
		"TCMALLOC_256K":             {"library/cpp/malloc/tcmalloc", "contrib/libs/tcmalloc"},
		"TCMALLOC_SMALL_BUT_SLOW":   {"library/cpp/malloc/tcmalloc", "contrib/libs/tcmalloc/small_but_slow"},
		"TCMALLOC_NUMA_256K":        {"library/cpp/malloc/tcmalloc", "contrib/libs/tcmalloc/numa_256k"},
		"TCMALLOC_NUMA_LARGE_PAGES": {"library/cpp/malloc/tcmalloc", "contrib/libs/tcmalloc/numa_large_pages"},
		"TCMALLOC":                  {"library/cpp/malloc/tcmalloc", "contrib/libs/tcmalloc/default"},
		"TCMALLOC_TC":               {"library/cpp/malloc/tcmalloc", "contrib/libs/tcmalloc/no_percpu_cache"},
		"GOOGLE":                    {"library/cpp/malloc/galloc"},
		"LF":                        {"library/cpp/lfalloc"},
		"LF_YT":                     {"library/cpp/lfalloc/yt"},
		"LF_DBG":                    {"library/cpp/lfalloc/dbg"},
		"B":                         {"library/cpp/balloc"},
		"BM":                        {"library/cpp/balloc_market"},
		"C":                         {"library/cpp/malloc/calloc"},
		"LOCKLESS":                  {"library/cpp/malloc/lockless"},
		"YT":                        {"library/cpp/ytalloc/impl"},
		"YT_TCMALLOC":               {"library/cpp/ytalloc/impl"},
		"YT_TCMALLOC_256K":          {"library/cpp/ytalloc/impl"},
		"FAKE":                      {},
	}

	withValgrind := bc.Flags["WITH_VALGRIND"] == "yes"
	sanitizerDefined := bc.Flags["SANITIZER_DEFINED"] == "yes"

	if (!bc.Musl && withValgrind) || sanitizerDefined {
		if allocType != "SYSTEM" {
			return []string{"library/cpp/malloc/system"}
		}
	}

	if bc.OS == "windows" && allocType == "J" {
		return []string{"library/cpp/malloc/system"}
	}

	if peers, ok := mappings[allocType]; ok {
		return peers
	}

	if allocType == "J" {
		return []string{"library/cpp/malloc/jemalloc"}
	}

	return []string{"library/cpp/malloc/system"}
}

func (bc *BuildConfig) ParseAllocatorFromModule(module *Module) string {
	if module == nil || module.Properties == nil {
		return bc.AllocatorType
	}

	if alloc, ok := module.Properties["ALLOCATOR"]; ok && alloc != "" {
		allocType := strings.TrimSpace(alloc)
		if bc.ValidAllocators[allocType] {
			return allocType
		}
	}

	return bc.AllocatorType
}

func (bc *BuildConfig) ValidateAllocator(allocType string) error {
	allocType = strings.TrimSpace(allocType)
	if !bc.ValidAllocators[allocType] {
		return fmt.Errorf("unknown allocator type: %s", allocType)
	}
	return nil
}

func detectOS(ctx *ParseContext) string {
	if ctx == nil || ctx.Platform == "" {
		return "linux"
	}

	platform := strings.ToLower(ctx.Platform)
	if strings.Contains(platform, "windows") || strings.Contains(platform, "win32") {
		return "windows"
	}
	if strings.Contains(platform, "darwin") || strings.Contains(platform, "macos") {
		return "darwin"
	}
	if strings.Contains(platform, "android") {
		return "android"
	}

	return "linux"
}

func detectArch(ctx *ParseContext) string {
	if ctx == nil || ctx.TargetPath == "" {
		return "x86_64"
	}

	target := strings.ToLower(ctx.TargetPath)
	if strings.Contains(target, "32") {
		return "32bit"
	}
	if strings.Contains(target, "aarch64") || strings.Contains(target, "arm64") {
		return "aarch64"
	}
	if strings.Contains(target, "xtensa") {
		return "xtensa"
	}

	return "x86_64"
}

func detectCompiler(ctx *ParseContext) string {
	if ctx == nil || ctx.Language == "" {
		return "gcc"
	}

	lang := strings.ToLower(ctx.Language)
	if strings.Contains(lang, "clang") || strings.Contains(lang, "llvm") {
		return "clang"
	}
	if strings.Contains(lang, "msvc") || strings.Contains(lang, "cl") {
		return "msvc"
	}

	return "gcc"
}
