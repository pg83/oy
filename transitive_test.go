package main

import (
	"testing"
)

func TestTransitiveSimpleChain(t *testing.T) {
	registry := NewModuleRegistry()

	dep3Module := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/base",
		Dependencies: []string{},
		Sources:      []string{"base.cpp"},
		Properties:   map[string]string{},
	}

	dep2Module := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/archive",
		Dependencies: []string{"library/cpp/base"},
		Sources:      []string{"archive.cpp"},
		Properties:   map[string]string{},
	}

	dep1Module := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/digest/md5",
		Dependencies: []string{"library/cpp/base"},
		Sources:      []string{"md5.cpp"},
		Properties:   map[string]string{},
	}

	mainModule := &Module{
		Type:         ModuleTypeProgram,
		SourcePath:   "tools/archiver",
		Dependencies: []string{"library/cpp/archive", "library/cpp/digest/md5"},
		Sources:      []string{"main.cpp"},
		Properties:   map[string]string{},
	}

	registry.Register(dep3Module.SourcePath, dep3Module)
	registry.Register(dep2Module.SourcePath, dep2Module)
	registry.Register(dep1Module.SourcePath, dep1Module)
	registry.Register(mainModule.SourcePath, mainModule)

	transitiveResolver := NewTransitiveResolver(registry)
	deps := transitiveResolver.ComputeTransitiveDeps(mainModule)

	expectedCount := 3
	if len(deps) != expectedCount {
		t.Errorf("expected %d transitive dependencies, got %d", expectedCount, len(deps))
	}

	expectedUID1 := NewUID([]byte(dep1Module.SourcePath))
	expectedUID2 := NewUID([]byte(dep2Module.SourcePath))
	expectedUID3 := NewUID([]byte(dep3Module.SourcePath))

	found1 := false
	found2 := false
	found3 := false
	for uid := range deps {
		if uid == expectedUID1 {
			found1 = true
		}
		if uid == expectedUID2 {
			found2 = true
		}
		if uid == expectedUID3 {
			found3 = true
		}
	}

	if !found1 || !found2 || !found3 {
		t.Errorf("expected to find UIDs %s, %s, and %s, got %v", expectedUID1, expectedUID2, expectedUID3, deps)
	}
}

func TestTransitiveDiamondDependency(t *testing.T) {
	registry := NewModuleRegistry()

	baseModule := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/base",
		Dependencies: []string{},
		Sources:      []string{"base.cpp"},
		Properties:   map[string]string{},
	}

	leftModule := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/left",
		Dependencies: []string{"library/cpp/base"},
		Sources:      []string{"left.cpp"},
		Properties:   map[string]string{},
	}

	rightModule := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/right",
		Dependencies: []string{"library/cpp/base"},
		Sources:      []string{"right.cpp"},
		Properties:   map[string]string{},
	}

	mainModule := &Module{
		Type:         ModuleTypeProgram,
		SourcePath:   "tools/app",
		Dependencies: []string{"library/cpp/left", "library/cpp/right"},
		Sources:      []string{"main.cpp"},
		Properties:   map[string]string{},
	}

	registry.Register(baseModule.SourcePath, baseModule)
	registry.Register(leftModule.SourcePath, leftModule)
	registry.Register(rightModule.SourcePath, rightModule)
	registry.Register(mainModule.SourcePath, mainModule)

	transitiveResolver := NewTransitiveResolver(registry)
	deps := transitiveResolver.ComputeTransitiveDeps(mainModule)

	expectedCount := 3
	if len(deps) != expectedCount {
		t.Errorf("expected %d transitive dependencies (diamond), got %d", expectedCount, len(deps))
	}

	baseUID := NewUID([]byte(baseModule.SourcePath))
	baseCount := 0
	for uid := range deps {
		if uid == baseUID {
			baseCount++
		}
	}

	if baseCount > 1 {
		t.Errorf("diamond dependency 'library/cpp/base' should appear only once, appeared %d times", baseCount)
	}
}

func TestTransitiveCycleDetection(t *testing.T) {
	registry := NewModuleRegistry()

	moduleA := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/a",
		Dependencies: []string{"library/b"},
		Sources:      []string{"a.cpp"},
		Properties:   map[string]string{},
	}

	moduleB := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/b",
		Dependencies: []string{"library/a"},
		Sources:      []string{"b.cpp"},
		Properties:   map[string]string{},
	}

	registry.Register(moduleA.SourcePath, moduleA)
	registry.Register(moduleB.SourcePath, moduleB)

	transitiveResolver := NewTransitiveResolver(registry)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for circular dependency")
		}
	}()

	transitiveResolver.ComputeTransitiveDeps(moduleA)
}

func TestTransitiveNoDependencies(t *testing.T) {
	registry := NewModuleRegistry()

	module := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "lib/standalone",
		Dependencies: []string{},
		Sources:      []string{"standalone.cpp"},
		Properties:   map[string]string{},
	}

	registry.Register(module.SourcePath, module)

	transitiveResolver := NewTransitiveResolver(registry)
	deps := transitiveResolver.ComputeTransitiveDeps(module)

	if len(deps) != 0 {
		t.Errorf("expected 0 transitive dependencies for module with no deps, got %d", len(deps))
	}
}
