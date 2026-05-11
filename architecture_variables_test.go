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
