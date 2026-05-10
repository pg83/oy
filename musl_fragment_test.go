package main

import (
	"os"
	"testing"
)

func TestParseModuleFragment(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"))
	module := ParseModuleFragment(string(content), "ya.make.inc")

	t.Logf("Module type: %v", module.Type)
	t.Logf("Sources: %d", len(module.Sources))
	t.Logf("Conditionals: %d", len(module.Conditionals))

	if len(module.Sources) > 0 {
		t.Logf("First 10 sources:")
		for i, src := range module.Sources {
			if i >= 10 {
				break
			}
			t.Logf("  %s", src)
		}
	}

	if len(module.Conditionals) > 0 {
		t.Logf("Conditionals: %d", len(module.Conditionals))
		for i, cond := range module.Conditionals {
			if i >= 1 {
				break
			}
			if cond.IfBranch != nil {
				t.Logf("  IF branch: %s with %d sources", cond.IfBranch.Condition, len(cond.IfBranch.Module.Sources))
			}
		}
	}
}
