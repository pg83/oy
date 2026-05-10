package main

import (
	"testing"
)

func TestDefaultAllocatorDetermination(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *ParseContext
		expectedAlloc string
	}{
		{
			name: "MUSL flag sets TCMALLOC_TC",
			ctx: &ParseContext{
				Musl: true,
			},
			expectedAlloc: "TCMALLOC_TC",
		},
		{
			name: "Windows platform uses J",
			ctx: &ParseContext{
				Platform: "windows",
			},
			expectedAlloc: "J",
		},
		{
			name: "Android platform uses J",
			ctx: &ParseContext{
				Platform: "linux-android",
			},
			expectedAlloc: "J",
		},
		{
			name: "32-bit architecture uses J",
			ctx: &ParseContext{
				TargetPath: "linux-32",
			},
			expectedAlloc: "J",
		},
		{
			name: "Linux x86_64 with clang uses TCMALLOC_TC",
			ctx: &ParseContext{
				Platform:   "linux",
				Language:   "clang",
				TargetPath: "x86_64",
			},
			expectedAlloc: "TCMALLOC_TC",
		},
		{
			name: "Default is SYSTEM",
			ctx: &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
			},
			expectedAlloc: "SYSTEM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewBuildConfig(tt.ctx)
			if config.AllocatorType != tt.expectedAlloc {
				t.Errorf("Expected allocator %s, got %s", tt.expectedAlloc, config.AllocatorType)
			}
		})
	}
}

func TestAllocatorPEERDIRMapping(t *testing.T) {
	tests := []struct {
		name           string
		allocator      string
		expectedDirs   []string
		configOverride func(*BuildConfig)
	}{
		{
			name:         "MIM maps to mimalloc",
			allocator:    "MIM",
			expectedDirs: []string{"library/cpp/malloc/mimalloc"},
		},
		{
			name:         "MIM_SDC maps to mimalloc_sdc",
			allocator:    "MIM_SDC",
			expectedDirs: []string{"library/cpp/malloc/mimalloc_sdc"},
		},
		{
			name:         "HU maps to hu",
			allocator:    "HU",
			expectedDirs: []string{"library/cpp/malloc/hu"},
		},
		{
			name:      "TCMALLOC TC maps to tcmalloc and no_percpu_cache",
			allocator: "TCMALLOC_TC",
			expectedDirs: []string{
				"library/cpp/malloc/tcmalloc",
				"contrib/libs/tcmalloc/no_percpu_cache",
			},
		},
		{
			name:      "TCMALLOC_256K maps to tcmalloc and default",
			allocator: "TCMALLOC_256K",
			expectedDirs: []string{
				"library/cpp/malloc/tcmalloc",
				"contrib/libs/tcmalloc",
			},
		},
		{
			name:         "GOOGLE maps to galloc",
			allocator:    "GOOGLE",
			expectedDirs: []string{"library/cpp/malloc/galloc"},
		},
		{
			name:         "LF maps to lfalloc",
			allocator:    "LF",
			expectedDirs: []string{"library/cpp/lfalloc"},
		},
		{
			name:         "B maps to balloc",
			allocator:    "B",
			expectedDirs: []string{"library/cpp/balloc"},
		},
		{
			name:         "LOCKLESS maps to lockless",
			allocator:    "LOCKLESS",
			expectedDirs: []string{"library/cpp/malloc/lockless"},
		},
		{
			name:         "YU maps to ytalloc",
			allocator:    "YT",
			expectedDirs: []string{"library/cpp/ytalloc/impl"},
		},
		{
			name:         "YT_TCMALLOC maps to ytalloc",
			allocator:    "YT_TCMALLOC",
			expectedDirs: []string{"library/cpp/ytalloc/impl"},
		},
		{
			name:         "YT_TCMALLOC_256K maps to ytalloc",
			allocator:    "YT_TCMALLOC_256K",
			expectedDirs: []string{"library/cpp/ytalloc/impl"},
		},
		{
			name:         "FAKE maps to empty dependencies",
			allocator:    "FAKE",
			expectedDirs: []string{},
		},
		{
			name:         "J maps to jemalloc on non-Windows",
			allocator:    "J",
			expectedDirs: []string{"library/cpp/malloc/jemalloc"},
		},
		{
			name:         "J maps to system on Windows",
			allocator:    "J",
			expectedDirs: []string{"library/cpp/malloc/system"},
			configOverride: func(bc *BuildConfig) {
				bc.OS = "windows"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
			}
			config := NewBuildConfig(ctx)

			if tt.configOverride != nil {
				tt.configOverride(config)
			}

			peerdirs := config.resolveAllocatorPEERDIRs(tt.allocator)

			if len(peerdirs) != len(tt.expectedDirs) {
				t.Errorf("Expected %d PEERDIRs, got %d", len(tt.expectedDirs), len(peerdirs))
			}

			for i, expected := range tt.expectedDirs {
				if i >= len(peerdirs) || peerdirs[i] != expected {
					t.Errorf("Expected PEERDIR %s at index %d, got %s", expected, i, safeGet(peerdirs, i))
				}
			}
		})
	}
}

func TestAllocatorInjectionInGraph(t *testing.T) {
	tests := []struct {
		name         string
		module       *Module
		ctx          *ParseContext
		expectInject bool
		expectDeps   []string
	}{
		{
			name: "PROGRAM module gets allocator dependencies",
			module: &Module{
				Type:         ModuleTypeProgram,
				SourcePath:   "test/program",
				Dependencies: []string{},
				Properties:   map[string]string{},
			},
			ctx: &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
			},
			expectInject: true,
			expectDeps:   []string{"library/cpp/malloc/system"},
		},
		{
			name: "ALLOCATOR() directive overrides default",
			module: &Module{
				Type:         ModuleTypeProgram,
				SourcePath:   "test/program",
				Dependencies: []string{},
				Properties:   map[string]string{"ALLOCATOR": "MIM"},
			},
			ctx: &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
			},
			expectInject: true,
			expectDeps:   []string{"library/cpp/malloc/mimalloc"},
		},
		{
			name: "FAKE allocator adds no dependencies",
			module: &Module{
				Type:         ModuleTypeProgram,
				SourcePath:   "test/program",
				Dependencies: []string{},
				Properties:   map[string]string{"ALLOCATOR": "FAKE"},
			},
			ctx: &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
			},
			expectInject: false,
			expectDeps:   []string{},
		},
		{
			name: "LIBRARY module does not get allocator dependencies",
			module: &Module{
				Type:         ModuleTypeLibrary,
				SourcePath:   "test/library",
				Dependencies: []string{},
				Properties:   map[string]string{},
			},
			ctx: &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
			},
			expectInject: false,
			expectDeps:   []string{},
		},
		{
			name: "Duplicate dependencies not added",
			module: &Module{
				Type:         ModuleTypeProgram,
				SourcePath:   "test/program",
				Dependencies: []string{"library/cpp/malloc/system"},
				Properties:   map[string]string{},
			},
			ctx: &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
			},
			expectInject: false,
			expectDeps:   []string{"library/cpp/malloc/system"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewBuildConfig(tt.ctx)
			builder := &GraphBuilder{
				config: config,
			}

			initialDepCount := len(tt.module.Dependencies)
			builder.injectAllocatorDependencies(tt.module)

			if tt.expectInject {
				if len(tt.module.Dependencies) <= initialDepCount {
					t.Error("Expected allocator dependencies to be injected")
				}
			} else {
				if len(tt.module.Dependencies) != initialDepCount {
					t.Error("Expected no allocator dependencies to be injected")
				}
			}

			for _, expectedDep := range tt.expectDeps {
				found := false
				for _, dep := range tt.module.Dependencies {
					if dep == expectedDep {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected dependency %s not found", expectedDep)
				}
			}
		})
	}
}

func TestMUSLDependencyInjection(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *ParseContext
		expectedAlloc string
		expectedDeps  []string
	}{
		{
			name: "--musl flag results in TCMALLOC_TC dependencies",
			ctx: &ParseContext{
				Musl:     true,
				Platform: "linux",
				Language: "gcc",
			},
			expectedAlloc: "TCMALLOC_TC",
			expectedDeps: []string{
				"library/cpp/malloc/tcmalloc",
				"contrib/libs/tcmalloc/no_percpu_cache",
			},
		},
		{
			name: "Without --musl flag uses default SYSTEM",
			ctx: &ParseContext{
				Musl:     false,
				Platform: "linux",
				Language: "gcc",
			},
			expectedAlloc: "SYSTEM",
			expectedDeps:  []string{"library/cpp/malloc/system"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewBuildConfig(tt.ctx)

			if config.AllocatorType != tt.expectedAlloc {
				t.Errorf("Expected allocator %s, got %s", tt.expectedAlloc, config.AllocatorType)
			}

			peerdirs := config.resolveAllocatorPEERDIRs(tt.expectedAlloc)
			if len(peerdirs) != len(tt.expectedDeps) {
				t.Errorf("Expected %d PEERDIRs, got %d", len(tt.expectedDeps), len(peerdirs))
			}

			for i, expected := range tt.expectedDeps {
				if i >= len(peerdirs) || peerdirs[i] != expected {
					t.Errorf("Expected PEERDIR %s at index %d, got %s", expected, i, safeGet(peerdirs, i))
				}
			}
		})
	}
}

func TestValidateAllocator(t *testing.T) {
	tests := []struct {
		name      string
		allocator string
		expectErr bool
	}{
		{
			name:      "Valid allocator MIM",
			allocator: "MIM",
			expectErr: false,
		},
		{
			name:      "Valid allocator TCMALLOC_TC",
			allocator: "TCMALLOC_TC",
			expectErr: false,
		},
		{
			name:      "Valid allocator FAKE",
			allocator: "FAKE",
			expectErr: false,
		},
		{
			name:      "Invalid allocator",
			allocator: "INVALID_ALLOCATOR",
			expectErr: true,
		},
		{
			name:      "Empty allocator",
			allocator: "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ParseContext{}
			config := NewBuildConfig(ctx)
			err := config.ValidateAllocator(tt.allocator)

			if tt.expectErr && err == nil {
				t.Error("Expected error for invalid allocator")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Did not expect error: %v", err)
			}
		})
	}
}

func safeGet(slice []string, index int) string {
	if index >= 0 && index < len(slice) {
		return slice[index]
	}
	return ""
}

func TestTestSafeGet(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		index    int
		expected string
	}{
		{
			name:     "Valid index",
			slice:    []string{"a", "b", "c"},
			index:    1,
			expected: "b",
		},
		{
			name:     "Index out of bounds negative",
			slice:    []string{"a", "b", "c"},
			index:    -1,
			expected: "",
		},
		{
			name:     "Index out of bounds too high",
			slice:    []string{"a", "b", "c"},
			index:    10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeGet(tt.slice, tt.index)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValgrindSanitizerOverride(t *testing.T) {
	tests := []struct {
		name         string
		allocator    string
		musl         bool
		withValgrind string
		sanitizerDef string
		expectedDirs []string
	}{
		{
			name:         "Valgrind without MUSL overrides to system",
			allocator:    "MIM",
			musl:         false,
			withValgrind: "yes",
			sanitizerDef: "",
			expectedDirs: []string{"library/cpp/malloc/system"},
		},
		{
			name:         "Sanitizer defined overrides to system",
			allocator:    "TCMALLOC_TC",
			musl:         false,
			withValgrind: "",
			sanitizerDef: "yes",
			expectedDirs: []string{"library/cpp/malloc/system"},
		},
		{
			name:         "Valgrind with MUSL does not override",
			allocator:    "MIM",
			musl:         true,
			withValgrind: "yes",
			sanitizerDef: "",
			expectedDirs: []string{"library/cpp/malloc/mimalloc"},
		},
		{
			name:         "SYSTEM allocator never overridden",
			allocator:    "SYSTEM",
			musl:         false,
			withValgrind: "yes",
			sanitizerDef: "yes",
			expectedDirs: []string{"library/cpp/malloc/system"},
		},
		{
			name:         "No override when neither condition met",
			allocator:    "TCMALLOC_TC",
			musl:         false,
			withValgrind: "no",
			sanitizerDef: "",
			expectedDirs: []string{
				"library/cpp/malloc/tcmalloc",
				"contrib/libs/tcmalloc/no_percpu_cache",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ParseContext{
				Platform:   "linux",
				Language:   "gcc",
				TargetPath: "x86_64",
				Musl:       tt.musl,
			}
			config := NewBuildConfig(ctx)
			config.Flags["WITH_VALGRIND"] = tt.withValgrind
			config.Flags["SANITIZER_DEFINED"] = tt.sanitizerDef

			peerdirs := config.resolveAllocatorPEERDIRs(tt.allocator)

			if len(peerdirs) != len(tt.expectedDirs) {
				t.Errorf("Expected %d PEERDIRs, got %d", len(tt.expectedDirs), len(peerdirs))
			}

			for i, expected := range tt.expectedDirs {
				if i >= len(peerdirs) || peerdirs[i] != expected {
					t.Errorf("Expected PEERDIR %s at index %d, got %s", expected, i, safeGet(peerdirs, i))
				}
			}
		})
	}
}
