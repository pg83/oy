package main

import (
	"os"
	"testing"
)

func TestConditionalResolveWithVars(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"))
	module := ParseModuleFragment(string(content), "ya.make.inc")

	t.Logf("Module conditionals: %d", len(module.Conditionals))

	buildContext := NewBuildContext()
	vars := NewMemoryVariableSet()
	bcVars := NewBuildContextVariableSet(buildContext, vars)

	t.Logf("ARCH_X86_64: %v", bcVars.IsTrue("ARCH_X86_64"))
	t.Logf("ARCH_AARCH64: %v", bcVars.IsTrue("ARCH_AARCH64"))

	resolvedModule := ResolveConditionals(module, buildContext, bcVars)

	t.Logf("Resolved sources: %d", len(resolvedModule.Sources))
	t.Logf("Resolved conditionals: %d", len(resolvedModule.Conditionals))

	if len(resolvedModule.Sources) > 0 {
		t.Logf("First 10 resolved sources:")
		for i, src := range resolvedModule.Sources {
			if i >= 10 {
				break
			}
			t.Logf("  %s", src)
		}
	}
}
