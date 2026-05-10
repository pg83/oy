package main

import (
	"testing"
)

func TestARNodeGenerationSingleSource(t *testing.T) {
	module := &Module{
		SourcePath:   "test/module",
		Type:         ModuleTypeLibrary,
		Sources:      []string{"main.c"},
		Dependencies: []string{},
		Properties:   make(map[string]string),
	}

	ctx := &ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: "test/module",
		BuildFlags: make(map[string]string),
	}

	registry := NewModuleRegistry()
	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/test/root")
	platformCtx := PlatformAwareContext{ctx: ctx, arch: PlatformX86_64}

	objectOutputs := []string{"$(BUILD_ROOT)/test/module/main.c.o"}

	arNode := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtx)

	if arNode == nil {
		t.Fatal("AR node should not be nil")
	}

	if arNode.KV["p"] != "AR" {
		t.Errorf("Expected KV.p = 'AR', got '%s'", arNode.KV["p"])
	}

	if arNode.KV["pc"] != "light-red" {
		t.Errorf("Expected KV.pc = 'light-red', got '%s'", arNode.KV["pc"])
	}

	if arNode.KV["show_out"] != "yes" {
		t.Errorf("Expected KV.show_out = 'yes', got '%s'", arNode.KV["show_out"])
	}

	if len(arNode.Cmds) != 1 {
		t.Errorf("Expected 1 command, got %d", len(arNode.Cmds))
	}

	if len(arNode.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(arNode.Outputs))
	}

	expectedOutput := "$(BUILD_ROOT)/test/module/libmodule.a"
	if arNode.Outputs[0] != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, arNode.Outputs[0])
	}

	hasLinkLibPy := false
	for _, input := range arNode.Inputs {
		if input == "$(SOURCE_ROOT)/build/scripts/link_lib.py" {
			hasLinkLibPy = true
			break
		}
	}
	if !hasLinkLibPy {
		t.Error("AR node should include link_lib.py in inputs")
	}
}

func TestARNodeGenerationMultipleSources(t *testing.T) {
	module := &Module{
		SourcePath:   "test/module",
		Type:         ModuleTypeLibrary,
		Sources:      []string{"main.c", "util.c", "helper.c"},
		Dependencies: []string{},
		Properties:   make(map[string]string),
	}

	ctx := &ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: "test/module",
		BuildFlags: make(map[string]string),
	}

	registry := NewModuleRegistry()
	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/test/root")
	platformCtx := PlatformAwareContext{ctx: ctx, arch: PlatformX86_64}

	objectOutputs := []string{
		"$(BUILD_ROOT)/test/module/main.c.o",
		"$(BUILD_ROOT)/test/module/util.c.o",
		"$(BUILD_ROOT)/test/module/helper.c.o",
	}

	arNode := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtx)

	if arNode == nil {
		t.Fatal("AR node should not be nil")
	}

	objectFileCount := 0
	for _, input := range arNode.Inputs {
		if len(input) > 2 && input[len(input)-2:] == ".o" {
			objectFileCount++
		}
	}

	if objectFileCount != 3 {
		t.Errorf("Expected 3 object files in inputs, got %d", objectFileCount)
	}

	sourceFileCount := 0
	for _, input := range arNode.Inputs {
		if len(input) > 2 && (input[len(input)-2:] == ".c" || input[len(input)-4:] == ".cpp") {
			sourceFileCount++
		}
	}

	if sourceFileCount != 3 {
		t.Errorf("Expected 3 source files in inputs, got %d", sourceFileCount)
	}
}

func TestARNodeCommandStructure(t *testing.T) {
	module := &Module{
		SourcePath:   "test/module",
		Type:         ModuleTypeLibrary,
		Sources:      []string{"main.c"},
		Dependencies: []string{},
		Properties:   make(map[string]string),
	}

	ctx := &ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: "test/module",
		BuildFlags: make(map[string]string),
	}

	registry := NewModuleRegistry()
	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/test/root")
	platformCtx := PlatformAwareContext{ctx: ctx, arch: PlatformX86_64}

	objectOutputs := []string{"$(BUILD_ROOT)/test/module/main.c.o"}

	arNode := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtx)

	if len(arNode.Cmds) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(arNode.Cmds))
	}

	cmdArgs := arNode.Cmds[0].CmdArgs

	if len(cmdArgs) < 9 {
		t.Fatalf("Expected at least 9 command args, got %d", len(cmdArgs))
	}

	if cmdArgs[0] != "$(YMAKE_PYTHON3-1002064631)/bin/python3" {
		t.Errorf("Expected first arg to be python3, got '%s'", cmdArgs[0])
	}

	if cmdArgs[1] != "$(SOURCE_ROOT)/build/scripts/link_lib.py" {
		t.Errorf("Expected second arg to be link_lib.py, got '%s'", cmdArgs[1])
	}

	if cmdArgs[2] != "$(CLANG-2403293607)/bin/llvm-ar" {
		t.Errorf("Expected third arg to be llvm-ar, got '%s'", cmdArgs[2])
	}

	if cmdArgs[3] != "LLVM_AR" {
		t.Errorf("Expected fourth arg to be LLVM_AR, got '%s'", cmdArgs[3])
	}

	if cmdArgs[4] != "gnu" {
		t.Errorf("Expected fifth arg to be gnu, got '%s'", cmdArgs[4])
	}
}

func TestARNodeEnvironmentVariables(t *testing.T) {
	module := &Module{
		SourcePath:   "test/module",
		Type:         ModuleTypeLibrary,
		Sources:      []string{"main.c"},
		Dependencies: []string{},
		Properties:   make(map[string]string),
	}

	ctx := &ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: "test/module",
		BuildFlags: make(map[string]string),
	}

	registry := NewModuleRegistry()
	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/test/root")
	platformCtx := PlatformAwareContext{ctx: ctx, arch: PlatformX86_64}

	objectOutputs := []string{"$(BUILD_ROOT)/test/module/main.c.o"}

	arNode := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtx)

	if len(arNode.Cmds) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(arNode.Cmds))
	}

	env := arNode.Cmds[0].Env

	if env["ARCADIA_ROOT_DISTBUILD"] != "$(SOURCE_ROOT)" {
		t.Errorf("Expected ARCADIA_ROOT_DISTBUILD = $(SOURCE_ROOT), got '%s'", env["ARCADIA_ROOT_DISTBUILD"])
	}

	if env["CPATH"] != "" {
		t.Errorf("Expected CPATH = '', got '%s'", env["CPATH"])
	}

	if env["DYLD_LIBRARY_PATH"] == "" {
		t.Error("Expected DYLD_LIBRARY_PATH to be set")
	}

	if env["LIBRARY_PATH"] != "" {
		t.Errorf("Expected LIBRARY_PATH = '', got '%s'", env["LIBRARY_PATH"])
	}

	if env["SDKROOT"] != "" {
		t.Errorf("Expected SDKROOT = '', got '%s'", env["SDKROOT"])
	}
}

func TestARNodeCompileDependencies(t *testing.T) {
	module := &Module{
		SourcePath:   "test/module",
		Type:         ModuleTypeLibrary,
		Sources:      []string{"main.c", "util.cpp"},
		Dependencies: []string{},
		Properties:   make(map[string]string),
	}

	ctx := &ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: "test/module",
		BuildFlags: make(map[string]string),
	}

	registry := NewModuleRegistry()
	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/test/root")
	platformCtx := PlatformAwareContext{ctx: ctx, arch: PlatformX86_64}

	objectOutputs := []string{
		"$(BUILD_ROOT)/test/module/main.c.o",
		"$(BUILD_ROOT)/test/module/util.cpp.o",
	}

	arNode := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtx)

	if len(arNode.Deps) != 2 {
		t.Errorf("Expected 2 compile dependencies, got %d", len(arNode.Deps))
	}

	for _, dep := range arNode.Deps {
		if dep == "" {
			t.Error("Compile dependency UID should not be empty")
		}
	}
}

func TestARNodePlatformSpecific(t *testing.T) {
	module := &Module{
		SourcePath:   "test/module",
		Type:         ModuleTypeLibrary,
		Sources:      []string{"main.c"},
		Dependencies: []string{},
		Properties:   make(map[string]string),
	}

	ctx := &ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: "test/module",
		BuildFlags: make(map[string]string),
	}

	registry := NewModuleRegistry()
	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/test/root")
	objectOutputs := []string{"$(BUILD_ROOT)/test/module/main.c.o"}

	platformCtxAARCH64 := PlatformAwareContext{ctx: ctx, arch: PlatformAARCH64}
	arNodeAARCH64 := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtxAARCH64)

	platformCtxX86_64 := PlatformAwareContext{ctx: ctx, arch: PlatformX86_64}
	arNodeX86_64 := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtxX86_64)

	if arNodeAARCH64.Platform != "default-linux-aarch64" {
		t.Errorf("Expected aarch64 node platform = 'default-linux-aarch64', got '%s'", arNodeAARCH64.Platform)
	}

	if arNodeX86_64.Platform != "default-linux-x86_64" {
		t.Errorf("Expected x86_64 node platform = 'default-linux-x86_64', got '%s'", arNodeX86_64.Platform)
	}

	if arNodeAARCH64.UID == arNodeX86_64.UID {
		t.Error("AR nodes for different platforms should have different UIDs")
	}

	aarch64Env := arNodeAARCH64.Cmds[0].Env
	x86_64Env := arNodeX86_64.Cmds[0].Env

	if aarch64Env["DYLD_LIBRARY_PATH"] == x86_64Env["DYLD_LIBRARY_PATH"] {
		t.Error("DYLD_LIBRARY_PATH should differ for different platforms")
	}
}

func TestARNodeTargetProperties(t *testing.T) {
	module := &Module{
		SourcePath:   "test/cpp_module",
		Type:         ModuleTypeLibrary,
		Sources:      []string{"main.cpp"},
		Dependencies: []string{},
		Properties:   make(map[string]string),
	}

	ctx := &ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: "test/cpp_module",
		BuildFlags: make(map[string]string),
	}

	registry := NewModuleRegistry()
	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/test/root")
	platformCtx := PlatformAwareContext{ctx: ctx, arch: PlatformX86_64}

	objectOutputs := []string{"$(BUILD_ROOT)/test/cpp_module/main.cpp.o"}

	arNode := gb.createARNode(module, objectOutputs, make(map[string]*Module), platformCtx)

	if arNode.TargetProperties.ModuleDir != "test/cpp_module" {
		t.Errorf("Expected ModuleDir = 'test/cpp_module', got '%s'", arNode.TargetProperties.ModuleDir)
	}

	if arNode.TargetProperties.ModuleLang != "cpp" {
		t.Errorf("Expected ModuleLang = 'cpp', got '%s'", arNode.TargetProperties.ModuleLang)
	}

	if arNode.TargetProperties.ModuleType != "lib" {
		t.Errorf("Expected ModuleType = 'lib', got '%s'", arNode.TargetProperties.ModuleType)
	}
}
