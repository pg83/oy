package main

import (
	"testing"
)

func TestBuildContextVariableSet_ARCHConstants(t *testing.T) {
	tests := []struct {
		name       string
		archString string
		arch       Arch
		wantX86    bool
		wantAARCH  bool
	}{
		{
			name:       "x86_64 architecture",
			archString: "x86_64",
			arch:       Arch64,
			wantX86:    true,
			wantAARCH:  false,
		},
		{
			name:       "aarch64 architecture",
			archString: "aarch64",
			arch:       Arch64,
			wantX86:    false,
			wantAARCH:  true,
		},
		{
			name:       "unknown architecture defaults to x86_64",
			archString: "unknown",
			arch:       Arch64,
			wantX86:    false,
			wantAARCH:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildContext := NewBuildContext()
			buildContext.ArchString = tt.archString
			buildContext.Arch = tt.arch

			vars := NewMemoryVariableSet()
			bcVars := NewBuildContextVariableSet(buildContext, vars)

			gotX86 := bcVars.IsTrue("ARCH_X86_64")
			gotAARCH := bcVars.IsTrue("ARCH_AARCH64")

			if gotX86 != tt.wantX86 {
				t.Errorf("ARCH_X86_64 = %v, want %v", gotX86, tt.wantX86)
			}

			if gotAARCH != tt.wantAARCH {
				t.Errorf("ARCH_AARCH64 = %v, want %v", gotAARCH, tt.wantAARCH)
			}
		})
	}
}

func TestBuildContextVariableSet_MutuallyExclusiveARCH(t *testing.T) {
	// Test that ARCH_X86_64 and ARCH_AARCH64 are mutually exclusive
	buildContext := NewBuildContext()
	vars := NewMemoryVariableSet()
	bcVars := NewBuildContextVariableSet(buildContext, vars)

	// Get default architecture from runtime
	archString := buildContext.ArchString

	x86True := bcVars.IsTrue("ARCH_X86_64")
	aarchTrue := bcVars.IsTrue("ARCH_AARCH64")

	// Only one should be true at a time
	if archString == "x86_64" {
		if !x86True {
			t.Errorf("ARCH_X86_64 should be true for x86_64 architecture")
		}
		if aarchTrue {
			t.Errorf("ARCH_AARCH64 should be false for x86_64 architecture")
		}
	} else if archString == "aarch64" {
		if x86True {
			t.Errorf("ARCH_X86_64 should be false for aarch64 architecture")
		}
		if !aarchTrue {
			t.Errorf("ARCH_AARCH64 should be true for aarch64 architecture")
		}
	} else {
		// For non-x86_64/non-aarch64 architectures, both should be false
		if x86True || aarchTrue {
			t.Errorf("Both ARCH_X86_64 and ARCH_AARCH64 should be false for architecture %s", archString)
		}
	}
}

func TestBuildContextVariableSet_ARCHHasKey(t *testing.T) {
	buildContext := NewBuildContext()
	vars := NewMemoryVariableSet()
	bcVars := NewBuildContextVariableSet(buildContext, vars)

	// Verify that ARCH constants are recognized keys
	if !bcVars.HasKey("ARCH_X86_64") {
		t.Errorf("BuildContextVariableSet should have key ARCH_X86_64")
	}

	if !bcVars.HasKey("ARCH_AARCH64") {
		t.Errorf("BuildContextVariableSet should have key ARCH_AARCH64")
	}

	if !bcVars.HasKey("ARCH_ARM64") {
		t.Errorf("BuildContextVariableSet should have key ARCH_ARM64")
	}

	if !bcVars.HasKey("ARCH_ARM6") {
		t.Errorf("BuildContextVariableSet should have key ARCH_ARM6")
	}

	if !bcVars.HasKey("ARCH_ARM7") {
		t.Errorf("BuildContextVariableSet should have key ARCH_ARM7")
	}

	if !bcVars.HasKey("ARCH_ARM") {
		t.Errorf("BuildContextVariableSet should have key ARCH_ARM")
	}

	if !bcVars.HasKey("ARCH_PPC64LE") {
		t.Errorf("BuildContextVariableSet should have key ARCH_PPC64LE")
	}

	if !bcVars.HasKey("ARCH_RISCV32") {
		t.Errorf("BuildContextVariableSet should have key ARCH_RISCV32")
	}

	if !bcVars.HasKey("ARCH_RISCV64") {
		t.Errorf("BuildContextVariableSet should have key ARCH_RISCV64")
	}
}

func TestBuildContextVariableSet_ARMARCHConstants(t *testing.T) {
	tests := []struct {
		name       string
		archString string
		arch       Arch
		wantARM64  bool
		wantARM6   bool
		wantARM7   bool
		wantARM    bool
	}{
		{
			name:       "aarch64 architecture",
			archString: "aarch64",
			arch:       Arch64,
			wantARM64:  true,
			wantARM6:   false,
			wantARM7:   false,
			wantARM:    true,
		},
		{
			name:       "arm64 architecture",
			archString: "arm64",
			arch:       Arch64,
			wantARM64:  true,
			wantARM6:   false,
			wantARM7:   false,
			wantARM:    true,
		},
		{
			name:       "armv6 architecture",
			archString: "armv6",
			arch:       Arch32,
			wantARM64:  false,
			wantARM6:   true,
			wantARM7:   false,
			wantARM:    true,
		},
		{
			name:       "armv7 architecture",
			archString: "armv7",
			arch:       Arch32,
			wantARM64:  false,
			wantARM6:   false,
			wantARM7:   true,
			wantARM:    true,
		},
		{
			name:       "x86_64 architecture",
			archString: "x86_64",
			arch:       Arch64,
			wantARM64:  false,
			wantARM6:   false,
			wantARM7:   false,
			wantARM:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildContext := NewBuildContext()
			buildContext.ArchString = tt.archString
			buildContext.Arch = tt.arch

			vars := NewMemoryVariableSet()
			bcVars := NewBuildContextVariableSet(buildContext, vars)

			gotARM64 := bcVars.IsTrue("ARCH_ARM64")
			gotARM6 := bcVars.IsTrue("ARCH_ARM6")
			gotARM7 := bcVars.IsTrue("ARCH_ARM7")
			gotARM := bcVars.IsTrue("ARCH_ARM")

			if gotARM64 != tt.wantARM64 {
				t.Errorf("ARCH_ARM64 = %v, want %v", gotARM64, tt.wantARM64)
			}

			if gotARM6 != tt.wantARM6 {
				t.Errorf("ARCH_ARM6 = %v, want %v", gotARM6, tt.wantARM6)
			}

			if gotARM7 != tt.wantARM7 {
				t.Errorf("ARCH_ARM7 = %v, want %v", gotARM7, tt.wantARM7)
			}

			if gotARM != tt.wantARM {
				t.Errorf("ARCH_ARM = %v, want %v", gotARM, tt.wantARM)
			}
		})
	}
}

func TestBuildContextVariableSet_RISCVAndPPCConstants(t *testing.T) {
	tests := []struct {
		name        string
		archString  string
		wantPPC64LE bool
		wantRISCV32 bool
		wantRISCV64 bool
	}{
		{
			name:        "ppc64le architecture",
			archString:  "ppc64le",
			wantPPC64LE: true,
			wantRISCV32: false,
			wantRISCV64: false,
		},
		{
			name:        "riscv32 architecture",
			archString:  "riscv32",
			wantPPC64LE: false,
			wantRISCV32: true,
			wantRISCV64: false,
		},
		{
			name:        "riscv64 architecture",
			archString:  "riscv64",
			wantPPC64LE: false,
			wantRISCV32: false,
			wantRISCV64: true,
		},
		{
			name:        "x86_64 architecture",
			archString:  "x86_64",
			wantPPC64LE: false,
			wantRISCV32: false,
			wantRISCV64: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildContext := NewBuildContext()
			buildContext.ArchString = tt.archString

			vars := NewMemoryVariableSet()
			bcVars := NewBuildContextVariableSet(buildContext, vars)

			gotPPC64LE := bcVars.IsTrue("ARCH_PPC64LE")
			gotRISCV32 := bcVars.IsTrue("ARCH_RISCV32")
			gotRISCV64 := bcVars.IsTrue("ARCH_RISCV64")

			if gotPPC64LE != tt.wantPPC64LE {
				t.Errorf("ARCH_PPC64LE = %v, want %v", gotPPC64LE, tt.wantPPC64LE)
			}

			if gotRISCV32 != tt.wantRISCV32 {
				t.Errorf("ARCH_RISCV32 = %v, want %v", gotRISCV32, tt.wantRISCV32)
			}

			if gotRISCV64 != tt.wantRISCV64 {
				t.Errorf("ARCH_RISCV64 = %v, want %v", gotRISCV64, tt.wantRISCV64)
			}
		})
	}
}

func TestBuildEngine_ArchStringPropagation(t *testing.T) {
	// Test that ParseContext.ArchString is correctly propagated to BuildContext.ArchString
	archStrings := []string{"x86_64", "aarch64"}

	for _, archStr := range archStrings {
		t.Run(archStr, func(t *testing.T) {
			parseCtx := ParseContext{
				ArchString: archStr,
			}

			sourceRoot := "/home/pg/monorepo/yatool_orig"
			engine := NewBuildEngine(&parseCtx, sourceRoot)

			buildContext := engine.createBuildContext()

			if buildContext.ArchString != archStr {
				t.Errorf("BuildContext.ArchString = %s, want %s", buildContext.ArchString, archStr)
			}

			vars := NewMemoryVariableSet()
			bcVars := NewBuildContextVariableSet(buildContext, vars)

			// Verify that the correct ARCH constant is true
			if archStr == "x86_64" {
				if !bcVars.IsTrue("ARCH_X86_64") {
					t.Errorf("ARCH_X86_64 should be true for %s", archStr)
				}
			} else if archStr == "aarch64" {
				if !bcVars.IsTrue("ARCH_AARCH64") {
					t.Errorf("ARCH_AARCH64 should be true for %s", archStr)
				}
			}
		})
	}
}
