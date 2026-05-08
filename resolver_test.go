package main

import (
	"testing"
)

func TestResolverDirectDependency(t *testing.T) {
	registry := NewModuleRegistry()
	ctx := &ParseContext{}

	depModule := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/archive",
		Dependencies: []string{},
		Sources:      []string{"archive.cpp"},
		Properties:   map[string]string{},
	}

	registry.Register(depModule.SourcePath, depModule)

	mainModule := &Module{
		Type:         ModuleTypeProgram,
		SourcePath:   "tools/archiver",
		Dependencies: []string{"library/cpp/archive"},
		Sources:      []string{"main.cpp"},
		Properties:   map[string]string{},
	}

	resolver := NewResolver(registry, ctx)
	resolved, err := resolver.ResolveDependencies(mainModule)
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	if len(resolved) != 1 {
		t.Errorf("expected 1 resolved dependency, got %d", len(resolved))
	}

	expectedUID := NewUID([]byte(depModule.SourcePath))
	if resolved[0] != expectedUID {
		t.Errorf("expected UID '%s', got '%s'", expectedUID, resolved[0])
	}
}

func TestResolverMultipleDependencies(t *testing.T) {
	registry := NewModuleRegistry()
	ctx := &ParseContext{}

	dep1Module := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/archive",
		Dependencies: []string{},
		Sources:      []string{"archive.cpp"},
		Properties:   map[string]string{},
	}

	dep2Module := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/digest/md5",
		Dependencies: []string{},
		Sources:      []string{"md5.cpp"},
		Properties:   map[string]string{},
	}

	registry.Register(dep1Module.SourcePath, dep1Module)
	registry.Register(dep2Module.SourcePath, dep2Module)

	mainModule := &Module{
		Type:         ModuleTypeProgram,
		SourcePath:   "tools/archiver",
		Dependencies: []string{"library/cpp/archive", "library/cpp/digest/md5"},
		Sources:      []string{"main.cpp"},
		Properties:   map[string]string{},
	}

	resolver := NewResolver(registry, ctx)
	resolved, err := resolver.ResolveDependencies(mainModule)
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	if len(resolved) != 2 {
		t.Errorf("expected 2 resolved dependencies, got %d", len(resolved))
	}

	expectedUID1 := NewUID([]byte(dep1Module.SourcePath))
	expectedUID2 := NewUID([]byte(dep2Module.SourcePath))

	found1 := false
	found2 := false
	for _, uid := range resolved {
		if uid == expectedUID1 {
			found1 = true
		}
		if uid == expectedUID2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("expected to find UIDs '%s' and '%s', got %v", expectedUID1, expectedUID2, resolved)
	}
}

func TestResolverMissingDependency(t *testing.T) {
	registry := NewModuleRegistry()
	ctx := &ParseContext{}

	mainModule := &Module{
		Type:         ModuleTypeProgram,
		SourcePath:   "tools/archiver",
		Dependencies: []string{"library/cpp/archive"},
		Sources:      []string{"main.cpp"},
		Properties:   map[string]string{},
	}

	resolver := NewResolver(registry, ctx)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for missing dependency")
		}
	}()

	resolver.ResolveDependencies(mainModule)
}

func TestResolverPathNormalization(t *testing.T) {
	registry := NewModuleRegistry()
	ctx := &ParseContext{}

	depModule := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "library/cpp/archive",
		Dependencies: []string{},
		Sources:      []string{"archive.cpp"},
		Properties:   map[string]string{},
	}

	registry.Register(depModule.SourcePath, depModule)

	mainModule := &Module{
		Type:         ModuleTypeProgram,
		SourcePath:   "tools/archiver",
		Dependencies: []string{"library/cpp/archive/"},
		Sources:      []string{"main.cpp"},
		Properties:   map[string]string{},
	}

	resolver := NewResolver(registry, ctx)
	resolved, err := resolver.ResolveDependencies(mainModule)
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	if len(resolved) != 1 {
		t.Errorf("expected 1 resolved dependency despite trailing slash, got %d", len(resolved))
	}
}
