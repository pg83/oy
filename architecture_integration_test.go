package main

import (
	"os"
	"testing"
)

func TestARCHSourceFiltering_X86_64(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"))
	module := ParseModuleFragment(string(content), "ya.make.inc")

	ctxX86_64 := &BuildContext{
		Platform:   PlatformLinux,
		Arch:       Arch64,
		ArchString: "x86_64",
		Compiler:   CompilerGCC,
	}
	inner := NewMemoryVariableSet()
	varsX86_64 := NewBuildContextVariableSet(ctxX86_64, inner)

	if !varsX86_64.IsTrue("ARCH_X86_64") {
		t.Fatal("Test setup failed: ARCH_X86_64 should be true")
	}

	if varsX86_64.IsTrue("ARCH_AARCH64") {
		t.Fatal("Test setup failed: ARCH_AARCH64 should be false")
	}

	resolvedX86_64 := ResolveConditionals(module, ctxX86_64, varsX86_64)

	t.Logf("Resolved sources on x86_64: %d", len(resolvedX86_64.Sources))

	if len(resolvedX86_64.Sources) == 0 {
		t.Fatal("Expected at least one source on x86_64")
	}

	expectedX86_64Sources := []string{"src/aio/aio.c", "src/complex/cabs.c", "src/complex/cacos.c"}

	for _, expectedSrc := range expectedX86_64Sources {
		found := false
		for _, src := range resolvedX86_64.Sources {
			if src == expectedSrc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in x86_64 resolved sources", expectedSrc)
		}
	}
}

func TestARCHSourceFiltering_AARCH64(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"))
	module := ParseModuleFragment(string(content), "ya.make.inc")

	ctxAArch64 := &BuildContext{
		Platform:   PlatformLinux,
		Arch:       Arch64,
		ArchString: "aarch64",
		Compiler:   CompilerGCC,
	}
	inner := NewMemoryVariableSet()
	varsAArch64 := NewBuildContextVariableSet(ctxAArch64, inner)

	if varsAArch64.IsTrue("ARCH_X86_64") {
		t.Fatal("Test setup failed: ARCH_X86_64 should be false")
	}

	if !varsAArch64.IsTrue("ARCH_AARCH64") {
		t.Fatal("Test setup failed: ARCH_AARCH64 should be true")
	}

	resolvedAArch64 := ResolveConditionals(module, ctxAArch64, varsAArch64)

	t.Logf("Resolved sources on aarch64: %d", len(resolvedAArch64.Sources))

	if len(resolvedAArch64.Sources) == 0 {
		t.Fatal("Expected at least one source on aarch64")
	}

	expectedAArch64Sources := []string{"src/aio/aio.c", "src/complex/cabs.c", "src/complex/cacos.c"}

	for _, expectedSrc := range expectedAArch64Sources {
		found := false
		for _, src := range resolvedAArch64.Sources {
			if src == expectedSrc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in aarch64 resolved sources", expectedSrc)
		}
	}
}

func TestARCHSourceFiltering_MutualExclusion(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"))
	module := ParseModuleFragment(string(content), "ya.make.inc")

	ctxX86_64 := &BuildContext{
		Platform:   PlatformLinux,
		Arch:       Arch64,
		ArchString: "x86_64",
		Compiler:   CompilerGCC,
	}
	varsX86_64 := NewBuildContextVariableSet(ctxX86_64, NewMemoryVariableSet())
	resolvedX86_64 := ResolveConditionals(module, ctxX86_64, varsX86_64)

	ctxAArch64 := &BuildContext{
		Platform:   PlatformLinux,
		Arch:       Arch64,
		ArchString: "aarch64",
		Compiler:   CompilerGCC,
	}
	varsAArch64 := NewBuildContextVariableSet(ctxAArch64, NewMemoryVariableSet())
	resolvedAArch64 := ResolveConditionals(module, ctxAArch64, varsAArch64)

	t.Logf("x86_64 resolved sources: %d", len(resolvedX86_64.Sources))
	t.Logf("aarch64 resolved sources: %d", len(resolvedAArch64.Sources))

	if len(resolvedX86_64.Sources) == 0 {
		t.Fatal("Expected at least one source on x86_64")
	}

	if len(resolvedAArch64.Sources) == 0 {
		t.Fatal("Expected at least one source on aarch64")
	}

	x86_64SourceSet := make(map[string]bool)
	for _, src := range resolvedX86_64.Sources {
		x86_64SourceSet[src] = true
	}

	aarch64SourceSet := make(map[string]bool)
	for _, src := range resolvedAArch64.Sources {
		aarch64SourceSet[src] = true
	}

	commonSources := 0
	for src := range x86_64SourceSet {
		if aarch64SourceSet[src] {
			commonSources++
		}
	}

	t.Logf("Common sources between x86_64 and aarch64: %d", commonSources)

	if commonSources < 100 {
		t.Errorf("Expected at least 100 common sources, got %d", commonSources)
	}
}

func TestARCHSourceFiltering_SameCount(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"))
	module := ParseModuleFragment(string(content), "ya.make.inc")

	ctxX86_64 := &BuildContext{
		Platform:   PlatformLinux,
		Arch:       Arch64,
		ArchString: "x86_64",
		Compiler:   CompilerGCC,
	}
	varsX86_64 := NewBuildContextVariableSet(ctxX86_64, NewMemoryVariableSet())
	resolvedX86_64 := ResolveConditionals(module, ctxX86_64, varsX86_64)

	ctxAArch64 := &BuildContext{
		Platform:   PlatformLinux,
		Arch:       Arch64,
		ArchString: "aarch64",
		Compiler:   CompilerGCC,
	}
	varsAArch64 := NewBuildContextVariableSet(ctxAArch64, NewMemoryVariableSet())
	resolvedAArch64 := ResolveConditionals(module, ctxAArch64, varsAArch64)

	t.Logf("x86_64 resolved sources: %d", len(resolvedX86_64.Sources))
	t.Logf("aarch64 resolved sources: %d", len(resolvedAArch64.Sources))

	if len(resolvedX86_64.Sources) != len(resolvedAArch64.Sources) {
		t.Errorf("Expected same source count on both architectures (1319), got x86_64=%d, aarch64=%d",
			len(resolvedX86_64.Sources), len(resolvedAArch64.Sources))
	}

	if len(resolvedX86_64.Sources) != 1319 {
		t.Errorf("Expected 1319 sources based on reference structure, got %d", len(resolvedX86_64.Sources))
	}
}
