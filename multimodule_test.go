package main

import (
	"testing"
)

func TestParseFlags_Musl(t *testing.T) {
	args := []string{"program", "--musl"}
	result := Throw2(ParseFlags(args))

	if !result.Ctx.Musl {
		t.Errorf("Expected Musl=true, got false")
	}

	if result.Ctx.TargetPlatform != "" {
		t.Errorf("Expected empty TargetPlatform, got %s", result.Ctx.TargetPlatform)
	}
}

func TestParseFlags_TargetPlatform(t *testing.T) {
	args := []string{"program", "--target-platform=default-linux-aarch64"}
	result := Throw2(ParseFlags(args))

	if result.Ctx.TargetPlatform != "default-linux-aarch64" {
		t.Errorf("Expected TargetPlatform=default-linux-aarch64, got %s", result.Ctx.TargetPlatform)
	}
}

func TestParseFlags_HostPlatformFlag(t *testing.T) {
	args := []string{"program", "--host-platform-flag=MUSL=yes"}
	result := Throw2(ParseFlags(args))

	val, ok := result.PlatformFlag.GetHostFlag("MUSL")
	if !ok {
		t.Errorf("Expected host flag MUSL to be set")
	}

	if val != "yes" {
		t.Errorf("Expected MUSL=yes, got %s", val)
	}
}

func TestParseFlags_TargetPlatformFlag(t *testing.T) {
	args := []string{"program", "--target-platform-flag=USE_ASMLIB=no"}
	result := Throw2(ParseFlags(args))

	val, ok := result.PlatformFlag.GetTargetFlag("USE_ASMLIB")
	if !ok {
		t.Errorf("Expected target flag USE_ASMLIB to be set")
	}

	if val != "no" {
		t.Errorf("Expected USE_ASMLIB=no, got %s", val)
	}
}

func TestParseFlags_Language(t *testing.T) {
	args := []string{"program", "--language=proto"}
	result := Throw2(ParseFlags(args))

	if result.Ctx.Language != "proto" {
		t.Errorf("Expected Language=proto, got %s", result.Ctx.Language)
	}
}

func TestParseFlags_Combined(t *testing.T) {
	args := []string{
		"program",
		"--musl",
		"--target-platform=default-linux-aarch64",
		"--host-platform-flag=MUSL=yes",
		"--target-platform-flag=USE_ASMLIB=no",
		"--language=go",
	}

	result := Throw2(ParseFlags(args))

	if !result.Ctx.Musl {
		t.Errorf("Expected Musl=true")
	}

	if result.Ctx.TargetPlatform != "default-linux-aarch64" {
		t.Errorf("Expected TargetPlatform=default-linux-aarch64, got %s", result.Ctx.TargetPlatform)
	}

	val, ok := result.PlatformFlag.GetHostFlag("MUSL")
	if !ok || val != "yes" {
		t.Errorf("Expected MUSL=yes")
	}

	val, ok = result.PlatformFlag.GetTargetFlag("USE_ASMLIB")
	if !ok || val != "no" {
		t.Errorf("Expected USE_ASMLIB=no")
	}

	if result.Ctx.Language != "go" {
		t.Errorf("Expected Language=go, got %s", result.Ctx.Language)
	}
}

func TestBuildContextVariableSet_MUSL(t *testing.T) {
	ctx := NewBuildContext()
	ctx.Musl = true

	inner := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(ctx, inner)

	if !vars.IsTrue("MUSL") {
		t.Errorf("Expected MUSL to be true")
	}

	ctx.Musl = false
	if vars.IsTrue("MUSL") {
		t.Errorf("Expected MUSL to be false")
	}
}

func TestBuildContextVariableSet_TargetPlatform(t *testing.T) {
	ctx := NewBuildContext()
	ctx.TargetPlatform = "default-linux-aarch64"

	inner := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(ctx, inner)

	if !vars.IsTrue("TARGET_PLATFORM_DEFAULT_LINUX_AARCH64") {
		t.Errorf("Expected TARGET_PLATFORM_DEFAULT_LINUX_AARCH64 to be true")
	}

	if vars.IsTrue("TARGET_PLATFORM_DEFAULT_LINUX_X86_64") {
		t.Errorf("Expected TARGET_PLATFORM_DEFAULT_LINUX_X86_64 to be false")
	}

	ctx.TargetPlatform = ""
	if vars.IsTrue("TARGET_PLATFORM_DEFAULT_LINUX_AARCH64") {
		t.Errorf("Expected TARGET_PLATFORM_DEFAULT_LINUX_AARCH64 to be false")
	}
}

func TestBuildContextVariableSet_Language(t *testing.T) {
	ctx := NewBuildContext()
	ctx.Language = "proto"

	inner := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(ctx, inner)

	if !vars.IsTrue("PROTO") {
		t.Errorf("Expected PROTO to be true")
	}

	if vars.IsTrue("GO") {
		t.Errorf("Expected GO to be false")
	}

	ctx.Language = "go"
	if !vars.IsTrue("GO") {
		t.Errorf("Expected GO to be true")
	}

	if vars.IsTrue("PROTO") {
		t.Errorf("Expected PROTO to be false")
	}
}

func TestApplyModuleFlags(t *testing.T) {
	module := &Module{
		EnabledFlags:  []string{"MUSL_LITE", "ENABLE_FEATURE_X"},
		DisabledFlags: []string{"USE_ASMLIB", "OLD_CODE"},
	}

	vars := NewMemoryVariableSet()

	ApplyModuleFlags(module, vars)

	if !vars.IsTrue("MUSL_LITE") {
		t.Errorf("Expected MUSL_LITE to be enabled")
	}

	if !vars.IsTrue("ENABLE_FEATURE_X") {
		t.Errorf("Expected ENABLE_FEATURE_X to be enabled")
	}

	if vars.IsTrue("USE_ASMLIB") {
		t.Errorf("Expected USE_ASMLIB to be disabled")
	}

	if vars.IsTrue("OLD_CODE") {
		t.Errorf("Expected OLD_CODE to be disabled")
	}
}

func TestPlatformFlags_ToMap(t *testing.T) {
	pf := NewPlatformFlags()
	pf.SetHostFlag("MUSL", "yes")
	pf.SetTargetFlag("USE_ASMLIB", "no")

	m := pf.ToMap()

	if len(m) != 2 {
		t.Errorf("Expected 2 flags in map, got %d", len(m))
	}

	if m["MUSL"] != "yes" {
		t.Errorf("Expected MUSL=yes, got %s", m["MUSL"])
	}

	if m["USE_ASMLIB"] != "no" {
		t.Errorf("Expected USE_ASMLIB=no, got %s", m["USE_ASMLIB"])
	}
}

func TestModule_AddEnabledFlag(t *testing.T) {
	module := &Module{}

	module.AddEnabledFlag("FLAG1")
	module.AddEnabledFlag("FLAG2")

	if len(module.EnabledFlags) != 2 {
		t.Errorf("Expected 2 enabled flags, got %d", len(module.EnabledFlags))
	}

	if module.EnabledFlags[0] != "FLAG1" {
		t.Errorf("Expected FLAG1, got %s", module.EnabledFlags[0])
	}

	if module.EnabledFlags[1] != "FLAG2" {
		t.Errorf("Expected FLAG2, got %s", module.EnabledFlags[1])
	}
}

func TestModule_AddDisabledFlag(t *testing.T) {
	module := &Module{}

	module.AddDisabledFlag("DIS1")
	module.AddDisabledFlag("DIS2")

	if len(module.DisabledFlags) != 2 {
		t.Errorf("Expected 2 disabled flags, got %d", len(module.DisabledFlags))
	}

	if module.DisabledFlags[0] != "DIS1" {
		t.Errorf("Expected DIS1, got %s", module.DisabledFlags[0])
	}

	if module.DisabledFlags[1] != "DIS2" {
		t.Errorf("Expected DIS2, got %s", module.DisabledFlags[1])
	}
}
