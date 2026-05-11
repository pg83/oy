package main

import (
	"testing"
)

func TestT154ArchConditionalsIntegration(t *testing.T) {
	sourceRoot := "/home/pg/monorepo/yatool_orig"
	target := "tools/archiver"

	ctx := ParseContext{
		Platform:   "linux",
		ArchString: "aarch64",
		TargetPath: target,
		Language:   "cpp",
		Musl:       true,
	}

	graph := Throw2(BuildDependencyGraph(target, ctx, sourceRoot, nil))

	t.Logf("Node count after T-154: %d", len(graph.Nodes))

	// Expected: closer to 3730 (reference) vs 3920 (before T-154)
	// The fix should reduce node count by excluding platform-specific sources
	if len(graph.Nodes) > 4000 {
		t.Logf("WARNING: Node count %d still high, expected ~3730", len(graph.Nodes))
	}

	t.Logf("Graph has %d nodes total", len(graph.Nodes))
}

func TestT154MuslArchConditionals(t *testing.T) {
	sourceRoot := "/home/pg/monorepo/yatool_orig"
	muslPath := sourceRoot + "/contrib/libs/musl/ya.make"

	file := ParseYaMakeFile(muslPath)

	ctxAARCH64 := NewBuildContext()
	ctxAARCH64.ArchString = "aarch64"
	ctxAARCH64.Platform = PlatformLinux

	varsAARCH64 := NewMemoryVariableSet()
	bcVarsAARCH64 := NewBuildContextVariableSet(ctxAARCH64, varsAARCH64)

	if len(file.Modules) == 0 {
		t.Fatalf("Expected at least 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]
	resolvedAARCH64 := ResolveConditionals(module, ctxAARCH64, bcVarsAARCH64)

	t.Logf("Musl module has %d sources for aarch64", len(resolvedAARCH64.Sources))

	ctxX86_64 := NewBuildContext()
	ctxX86_64.ArchString = "x86_64"
	ctxX86_64.Platform = PlatformLinux

	varsX86_64 := NewMemoryVariableSet()
	bcVarsX86_64 := NewBuildContextVariableSet(ctxX86_64, varsX86_64)

	resolvedX86_64 := ResolveConditionals(module, ctxX86_64, bcVarsX86_64)

	t.Logf("Musl module has %d sources for x86_64", len(resolvedX86_64.Sources))
}

func TestT154LibcxxArchConditionals(t *testing.T) {
	sourceRoot := "/home/pg/monorepo/yatool_orig"
	libcxxPath := sourceRoot + "/contrib/libs/cxxsupp/libcxx/ya.make"

	file := ParseYaMakeFile(libcxxPath)

	ctxARM := NewBuildContext()
	ctxARM.ArchString = "armv7"
	ctxARM.Platform = PlatformLinux

	varsARM := NewMemoryVariableSet()
	bcVarsARM := NewBuildContextVariableSet(ctxARM, varsARM)

	if len(file.Modules) == 0 {
		t.Fatalf("Expected at least 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	t.Logf("ARCH_ARM6: %v", bcVarsARM.IsTrue("ARCH_ARM6"))
	t.Logf("ARCH_ARM7: %v", bcVarsARM.IsTrue("ARCH_ARM7"))
	t.Logf("ARCH_ARM: %v", bcVarsARM.IsTrue("ARCH_ARM"))

	resolvedARM := ResolveConditionals(module, ctxARM, bcVarsARM)

	t.Logf("Libcxx module has %d sources for armv7", len(resolvedARM.Sources))
}
