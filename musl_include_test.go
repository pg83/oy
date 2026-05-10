package main

import (
	"os"
	"testing"
)

func TestBuildEngineProcessInclude(t *testing.T) {
	content := Throw2(os.ReadFile("/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make"))
	file := ParseYaMakeString(string(content), "ya.make")

	if len(file.Modules) == 0 {
		t.Fatal("No modules found")
	}

	module := file.Modules[0]

	t.Logf("IncludeDirectives: %d", len(module.IncludeDirectives))

	for i, include := range module.IncludeDirectives {
		t.Logf("Include %d: %s", i, include.FilePath)
	}

	buildContext := NewBuildContext()
	resolvedModule := ResolveConditionals(module, buildContext, NewMemoryVariableSet())

	t.Logf("Resolved IncludeDirectives: %d", len(resolvedModule.IncludeDirectives))

	for _, include := range resolvedModule.IncludeDirectives {
		resolvedPath := "/home/pg/monorepo/yatool_orig/contrib/libs/musl/" + include.FilePath
		includeContent := Throw2(os.ReadFile(resolvedPath))

		includeFile := ParseYaMakeString(string(includeContent), resolvedPath)

		if len(includeFile.Modules) > 0 {
			t.Logf("Include file has %d modules", len(includeFile.Modules))
		} else {
			t.Logf("Include file has no modules, trying fragment parsing")
			inclModule := ParseModuleFragment(string(includeContent), resolvedPath)
			t.Logf("Fragment module has %d sources, %d conditionals", len(inclModule.Sources), len(inclModule.Conditionals))

			resolvedInclModule := ResolveConditionals(inclModule, buildContext, NewMemoryVariableSet())
			t.Logf("Resolved fragment has %d sources", len(resolvedInclModule.Sources))
		}
	}
}
