package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYaMakeInc(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"))

	parser := NewParser(string(content), filepath.Join("/", "home", "pg", "monorepo", "yatool_orig", "contrib", "libs", "musl", "ya.make.inc"))
	file := parser.Parse()

	t.Logf("Modules: %d", len(file.Modules))
	t.Logf("Imports: %d", len(file.Imports))

	if len(file.Modules) > 0 {
		t.Logf("First module sources: %d", len(file.Modules[0].Sources))
		t.Logf("First module type: %v", file.Modules[0].Type)

		if len(file.Modules[0].Sources) > 0 {
			t.Logf("First 10 sources:")
			for i, src := range file.Modules[0].Sources {
				if i >= 10 {
					break
				}
				t.Logf("  %s", src)
			}
		}

		if len(file.Modules[0].Conditionals) > 0 {
			t.Logf("First module conditionals: %d", len(file.Modules[0].Conditionals))
			for i, cond := range file.Modules[0].Conditionals {
				if i >= 2 {
					break
				}
				if cond.IfBranch != nil {
					t.Logf("  IF branch: %s with %d sources", cond.IfBranch.Condition, len(cond.IfBranch.Module.Sources))
				}
			}
		}
	}
}

func TestArchitectureDetection(t *testing.T) {
	ctx := NewBuildContext()
	t.Logf("Architecture string: %s", ctx.ArchString)
	t.Logf("Arch enum: %v", ctx.Arch)

	vars := NewMemoryVariableSet()
	bcVars := NewBuildContextVariableSet(ctx, vars)

	t.Logf("ARCH_X86_64: %v", bcVars.IsTrue("ARCH_X86_64"))
	t.Logf("ARCH_AARCH64: %v", bcVars.IsTrue("ARCH_AARCH64"))
}
