package main

import (
	"testing"
)

func TestNormalizedPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"library/cpp/archive", "library/cpp/archive"},
		{"/library/cpp/archive", "library/cpp/archive"},
		{"library/cpp/archive/", "library/cpp/archive"},
		{"/library/cpp/archive/", "library/cpp/archive"},
		{"tools/archiver", "tools/archiver"},
		{"/tools/archiver", "tools/archiver"},
		{"simple", "simple"},
		{"/simple/", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizedPath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizedPath(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewModuleRegistry(t *testing.T) {
	reg := NewModuleRegistry()

	if reg == nil {
		t.Fatal("NewModuleRegistry() returned nil")
	}

	if reg.Count() != 0 {
		t.Errorf("NewModuleRegistry().Count() = %d, expected 0", reg.Count())
	}

	if reg.List() == nil {
		t.Fatal("NewModuleRegistry().List() returned nil")
	}

	if len(reg.List()) != 0 {
		t.Errorf("NewModuleRegistry().List() length = %d, expected 0", len(reg.List()))
	}
}

func TestModuleRegistryRegisterAndGet(t *testing.T) {
	reg := NewModuleRegistry()

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/archive",
	}

	reg.Register("library/cpp/archive", module)

	if reg.Count() != 1 {
		t.Errorf("registry.Count() = %d, expected 1", reg.Count())
	}

	retrieved := reg.Get("library/cpp/archive")

	if retrieved == nil {
		t.Fatal("registry.Get() returned nil")
	}

	if retrieved != module {
		t.Error("registry.Get() returned different Module instance")
	}

	list := reg.List()

	if len(list) != 1 {
		t.Errorf("registry.List() length = %d, expected 1", len(list))
	}

	if list[0] != "library/cpp/archive" {
		t.Errorf("registry.List()[0] = %q, expected %q", list[0], "library/cpp/archive")
	}
}

func TestModuleRegistryPathNormalization(t *testing.T) {
	reg := NewModuleRegistry()

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/archive",
	}

	reg.Register("/library/cpp/archive/", module)

	if reg.Count() != 1 {
		t.Errorf("registry.Count() = %d, expected 1", reg.Count())
	}

	if !reg.Has("library/cpp/archive") {
		t.Error("registry.Has(\"library/cpp/archive\") = false, expected true")
	}

	if !reg.Has("/library/cpp/archive") {
		t.Error("registry.Has(\"/library/cpp/archive\") = false, expected true")
	}

	if !reg.Has("library/cpp/archive/") {
		t.Error("registry.Has(\"library/cpp/archive/\") = false, expected true")
	}

	if reg.Has("library/cpp.archive/") {
		t.Error("registry.Has() with different path (typo) returned true, expected false")
	}
}

func TestModuleRegistryReplace(t *testing.T) {
	reg := NewModuleRegistry()

	module1 := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/archive",
	}

	module2 := &Module{
		Type:       ModuleTypeProgram,
		SourcePath: "library/cpp/archive",
	}

	reg.Register("library/cpp/archive", module1)

	if reg.Get("library/cpp/archive") != module1 {
		t.Error("Failed to register first module")
	}

	reg.Register("library/cpp/archive", module2)

	if reg.Get("library/cpp/archive") != module2 {
		t.Error("Failed to replace module")
	}

	if reg.Count() != 1 {
		t.Errorf("registry.Count() = %d, expected 1 after replacement", reg.Count())
	}
}

func TestModuleRegistryHas(t *testing.T) {
	reg := NewModuleRegistry()

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/archive",
	}

	if reg.Has("library/cpp/archive") {
		t.Error("registry.Has() returned true for unregistered module")
	}

	reg.Register("library/cpp/archive", module)

	if !reg.Has("library/cpp/archive") {
		t.Error("registry.Has() returned false for registered module")
	}

	if reg.Has("nonexistent/module") {
		t.Error("registry.Has() returned true for nonexistent path")
	}
}

func TestModuleRegistryRemove(t *testing.T) {
	reg := NewModuleRegistry()

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/archive",
	}

	reg.Register("library/cpp/archive", module)

	if reg.Count() != 1 {
		t.Errorf("registry.Count() = %d, expected 1", reg.Count())
	}

	reg.Remove("library/cpp/archive")

	if reg.Count() != 0 {
		t.Errorf("registry.Count() = %d, expected 0 after Remove", reg.Count())
	}

	if reg.Has("library/cpp/archive") {
		t.Error("registry.Has() returned true after Remove")
	}

	if reg.Get("library/cpp/archive") != nil {
		t.Error("registry.Get() returned non-nil after Remove")
	}

	reg.Remove("nonexistent/module")

	if reg.Count() != 0 {
		t.Errorf("registry.Count() = %d, expected 0 after removing nonexistent", reg.Count())
	}
}

func TestModuleRegistryNilRegistry(t *testing.T) {
	var reg *ModuleRegistry = nil

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/archive",
	}

	reg.Register("library/cpp/archive", module)

	if reg.Get("library/cpp/archive") != nil {
		t.Error("nil registry.Get() returned non-nil")
	}

	if reg.Has("library/cpp/archive") {
		t.Error("nil registry.Has() returned true")
	}

	if reg.Count() != 0 {
		t.Errorf("nil registry.Count() = %d, expected 0", reg.Count())
	}

	reg.Remove("library/cpp/archive")

	if reg.List() != nil {
		t.Error("nil registry.List() returned non-nil")
	}
}

func TestModuleRegisterNilModule(t *testing.T) {
	reg := NewModuleRegistry()

	reg.Register("library/cpp/archive", nil)

	if reg.Count() != 0 {
		t.Errorf("registry.Count() = %d, expected 0 after registering nil module", reg.Count())
	}

	if reg.Has("library/cpp/archive") {
		t.Error("registry.Has() returned true after registering nil module")
	}
}

func TestModuleRegistryAllModules(t *testing.T) {
	reg := NewModuleRegistry()

	archiveModule := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/archive",
	}

	md5Module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "library/cpp/digest/md5",
	}

	reg.Register("library/cpp/archive", archiveModule)
	reg.Register("library/cpp/digest/md5", md5Module)

	allModules := reg.AllModules()

	if len(allModules) != 2 {
		t.Errorf("registry.AllModules() length = %d, expected 2", len(allModules))
	}

	found := make(map[string]bool)
	for _, m := range allModules {
		if m.SourcePath == "library/cpp/archive" || m.SourcePath == "library/cpp/digest/md5" {
			found[m.SourcePath] = true
		}
	}

	if !found["library/cpp/archive"] {
		t.Error("archive module not found in AllModules()")
	}

	if !found["library/cpp/digest/md5"] {
		t.Error("md5 module not found in AllModules()")
	}
}

func TestModuleRegistryListCopy(t *testing.T) {
	reg := NewModuleRegistry()

	reg.Register("module1", &Module{SourcePath: "module1"})
	reg.Register("module2", &Module{SourcePath: "module2"})

	list1 := reg.List()
	list2 := reg.List()

	if len(list1) != len(list2) {
		t.Errorf("List() returned different lengths: %d vs %d", len(list1), len(list2))
	}

	list1[0] = "modified"

	if list2[0] == "modified" {
		t.Error("List() did not return a copy, modifications affected internal state")
	}

	if list2[0] == "module1" && list1[0] == "modified" {
		copyWorks := false
		for _, s := range list2 {
			if s == "module1" {
				copyWorks = true
				break
			}
		}
		if !copyWorks {
			t.Error("List() modifications leaked to subsequent calls")
		}
	}
}

func TestModuleRegistryConcurrency(t *testing.T) {
	reg := NewModuleRegistry()

	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func(n int) {
			path := "module/" + string(rune('0'+n))
			module := &Module{
				Type:       ModuleTypeLibrary,
				SourcePath: path,
			}

			reg.Register(path, module)
			reg.Get(path)
			reg.Has(path)
			reg.List()
			reg.AllModules()

			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

}

func TestModuleRegistryComplexPaths(t *testing.T) {
	reg := NewModuleRegistry()

	paths := []string{
		"library/cpp/archive",
		"/library/cpp/digest/md5/",
		"tools/archiver",
		"/simple/",
		"a/b/c/d/e",
	}

	modules := make([]*Module, len(paths))
	for i, p := range paths {
		modules[i] = &Module{
			Type:       ModuleTypeLibrary,
			SourcePath: p,
		}
		reg.Register(p, modules[i])
	}

	expectedCount := 5
	if reg.Count() != expectedCount {
		t.Errorf("registry.Count() = %d, expected %d", reg.Count(), expectedCount)
	}

	allModules := reg.AllModules()
	if len(allModules) != expectedCount {
		t.Errorf("registry.AllModules() length = %d, expected %d", len(allModules), expectedCount)
	}

	for _, p := range paths {
		if !reg.Has(p) {
			t.Errorf("registry.Has(%q) returned false", p)
		}

		retrieved := reg.Get(p)
		if retrieved == nil {
			t.Errorf("registry.Get(%q) returned nil", p)
		}
	}
}
