package main

import (
	"testing"
)

func TestBuiltinsConditionalBranchSelection(t *testing.T) {
	sourceRoot := "/home/pg/monorepo/yatool_orig"
	builtinsPath := sourceRoot + "/contrib/libs/cxxsupp/builtins/ya.make"

	file := ParseYaMakeFile(builtinsPath)

	if len(file.Modules) != 1 {
		t.Fatalf("Expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	ctxAARCH64 := NewBuildContext()
	ctxAARCH64.ArchString = "aarch64"
	ctxAARCH64.Platform = PlatformLinux

	varsAARCH64 := NewMemoryVariableSet()
	bcVarsAARCH64 := NewBuildContextVariableSet(ctxAARCH64, varsAARCH64)

	t.Logf("=== Testing AARCH64 context ===")
	t.Logf("ARCH_AARCH64: %v", bcVarsAARCH64.IsTrue("ARCH_AARCH64"))
	t.Logf("ARCH_ARM6: %v", bcVarsAARCH64.IsTrue("ARCH_ARM6"))
	t.Logf("ARCH_ARM7: %v", bcVarsAARCH64.IsTrue("ARCH_ARM7"))
	t.Logf("ARCH_ARM: %v", bcVarsAARCH64.IsTrue("ARCH_ARM"))
	t.Logf("ARCH_X86_64: %v", bcVarsAARCH64.IsTrue("ARCH_X86_64"))
	t.Logf("ARCH_ARM64: %v", bcVarsAARCH64.IsTrue("ARCH_ARM64"))

	t.Logf("Module has %d conditional blocks", len(module.Conditionals))
	for i, block := range module.Conditionals {
		t.Logf("Conditional block %d: IF=%s, %d ELSEIFs, ELSE=%v",
			i, block.IfBranch.Condition, len(block.ElseIfs), block.ElseBranch != nil)
		for j, elseif := range block.ElseIfs {
			t.Logf("  ELSEIF %d: %s", j, elseif.Condition)
		}
	}

	resolvedAARCH64 := ResolveConditionals(module, ctxAARCH64, bcVarsAARCH64)
	t.Logf("Resolved AARCH64 module has %d sources", len(resolvedAARCH64.Sources))

	aarch64Sources := []string{
		"aarch64/chkstk.S",
		"aarch64/fp_mode.c",
		"aarch64/sme-abi-assert.c",
	}

	for _, src := range aarch64Sources {
		found := false
		for _, s := range resolvedAARCH64.Sources {
			if s == src {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("aarch64 context: expected source %s not found in resolved sources", src)
		}
	}

	x86_64Sources := []string{
		"cpu_model/x86.c",
		"x86_64/chkstk.S",
		"x86_64/floatdidf.c",
	}

	for _, src := range x86_64Sources {
		for _, s := range resolvedAARCH64.Sources {
			if s == src {
				t.Errorf("aarch64 context: x86_64-specific source %s should not be present", src)
				break
			}
		}
	}

	ctxX86_64 := NewBuildContext()
	ctxX86_64.ArchString = "x86_64"
	ctxX86_64.Platform = PlatformLinux

	varsX86_64 := NewMemoryVariableSet()
	bcVarsX86_64 := NewBuildContextVariableSet(ctxX86_64, varsX86_64)

	resolvedX86_64 := ResolveConditionals(module, ctxX86_64, bcVarsX86_64)

	for _, src := range x86_64Sources {
		found := false
		for _, s := range resolvedX86_64.Sources {
			if s == src {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("x86_64 context: expected source %s not found in resolved sources", src)
		}
	}

	for _, src := range aarch64Sources {
		for _, s := range resolvedX86_64.Sources {
			if s == src {
				t.Errorf("x86_64 context: aarch64-specific source %s should not be present", src)
				break
			}
		}
	}
}

func TestBuiltinsArchConditionalEvaluation(t *testing.T) {
	sourceRoot := "/home/pg/monorepo/yatool_orig"
	builtinsPath := sourceRoot + "/contrib/libs/cxxsupp/builtins/ya.make"

	file := ParseYaMakeFile(builtinsPath)

	if len(file.Modules) != 1 {
		t.Fatalf("Expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	ctx := NewBuildContext()
	ctx.ArchString = "aarch64"
	ctx.Platform = PlatformLinux

	vars := NewMemoryVariableSet()
	bcVars := NewBuildContextVariableSet(ctx, vars)

	resolved := ResolveConditionals(module, ctx, bcVars)

	vars.SetValue("ARCH_AARCH64", "yes")

	if bcVars.IsTrue("ARCH_AARCH64") {
		t.Logf("ARCH_AARCH64 is true for aarch64 context - correct")
	} else {
		t.Errorf("ARCH_AARCH64 should be true for aarch64 context")
	}

	if bcVars.IsTrue("ARCH_ARM6") {
		t.Errorf("ARCH_ARM6 should be false for aarch64 context")
	}

	if bcVars.IsTrue("ARCH_ARM7") {
		t.Errorf("ARCH_ARM7 should be false for aarch64 context")
	}

	if !bcVars.IsTrue("ARCH_ARM64") {
		t.Errorf("ARCH_ARM64 should be true for aarch64 context")
	}

	t.Logf("Resolved builtins module has %d sources for aarch64", len(resolved.Sources))
}
