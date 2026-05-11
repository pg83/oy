package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPlatformContextInEvalTraces(t *testing.T) {
	ctx := NewBuildContext()
	vars := NewBuildContextVariableSet(ctx, NewMemoryVariableSet())

	logger := NewTraversalLogger(true, true, false)
	SetGlobalTraversalLogger(logger)

	module := &Module{
		SourcePath: "test/module",
		NoPlatform: false,
	}

	ApplyModuleFlags(module, vars)

	ast := ParseConditionExpression("MUSL")
	if ast == nil {
		t.Fatal("ParseConditionExpression returned nil")
	}

	eval := NewEvaluator(vars)
	SetEvaluatorContext(eval, "IF", "test/module")
	SetEvaluatorNoPlatform(eval, module.NoPlatform)
	result := eval.Evaluate(ast)

	if len(logger.evalVarTraces) != 1 {
		t.Fatalf("Got %d traces, want 1", len(logger.evalVarTraces))
	}

	trace := logger.evalVarTraces[0]
	if trace.ArchValue != ctx.ArchString {
		t.Errorf("ArchValue = %s, want %s", trace.ArchValue, ctx.ArchString)
	}
	if trace.MuslValue != ctx.Musl {
		t.Errorf("MuslValue = %v, want %v", trace.MuslValue, ctx.Musl)
	}
	if trace.OSValue != vars.GetOSValue() {
		t.Errorf("OSValue = %s, want %s", trace.OSValue, vars.GetOSValue())
	}
	if trace.NoPlatformValue != module.NoPlatform {
		t.Errorf("NoPlatformValue = %v, want %v", trace.NoPlatformValue, module.NoPlatform)
	}
	if trace.VariableName != "MUSL" {
		t.Errorf("VariableName = %s, want MUSL", trace.VariableName)
	}
	if result != ctx.Musl {
		t.Errorf("Result = %v, want %v", result, ctx.Musl)
	}
}

func TestPlatformContextSummaryOutput(t *testing.T) {
	ctx := NewBuildContext()
	vars := NewBuildContextVariableSet(ctx, NewMemoryVariableSet())
	vars.SetValue("TEST_VAR", "yes")

	logger := NewTraversalLogger(true, true, false)
	SetGlobalTraversalLogger(logger)

	testModule := &Module{
		SourcePath: "test",
		NoPlatform: false,
	}

	ApplyModuleFlags(testModule, vars)

	testCases := []string{
		"MUSL",
		"OS_LINUX",
		"ARCH_X86_64",
	}

	for _, tc := range testCases {
		ast := ParseConditionExpression(tc)
		if ast == nil {
			t.Fatalf("ParseConditionExpression(%q) returned nil", tc)
		}
		eval := NewEvaluator(vars)
		SetEvaluatorContext(eval, "IF", "test")
		SetEvaluatorNoPlatform(eval, testModule.NoPlatform)
		eval.Evaluate(ast)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger.OutputSummary()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	outputStr := buf.String()

	if !strings.Contains(outputStr, "Platform Variable Distribution") {
		t.Errorf("Output missing platform distribution section")
	}
	if !strings.Contains(outputStr, "ARCH-related evaluations:") {
		t.Errorf("Output missing ARCH-related evaluations count")
	}
	if !strings.Contains(outputStr, "MUSL-related evaluations:") {
		t.Errorf("Output missing MUSL-related evaluations count")
	}
	if !strings.Contains(outputStr, "OS-related evaluations:") {
		t.Errorf("Output missing OS-related evaluations count")
	}
	if !strings.Contains(outputStr, "NO_PLATFORM-related evaluations:") {
		t.Errorf("Output missing NO_PLATFORM-related evaluations count")
	}
	if !strings.Contains(outputStr, "Platform variable evaluations by result") {
		t.Errorf("Output missing platform variable result summary")
	}
	if !strings.Contains(outputStr, "arch=") {
		t.Errorf("Output missing arch context in trace details")
	}
	if !strings.Contains(outputStr, "musl=") {
		t.Errorf("Output missing musl context in trace details")
	}
	if !strings.Contains(outputStr, "os=") {
		t.Errorf("Output missing os context in trace details")
	}
}

func TestNoPlatformContextInEvalTraces(t *testing.T) {
	ctx := NewBuildContext()
	vars := NewBuildContextVariableSet(ctx, NewMemoryVariableSet())

	logger := NewTraversalLogger(true, true, false)
	SetGlobalTraversalLogger(logger)

	module := &Module{
		SourcePath: "test/module",
		NoPlatform: true,
	}

	ApplyModuleFlags(module, vars)

	ast := ParseConditionExpression("MUSL")
	if ast == nil {
		t.Fatal("ParseConditionExpression returned nil")
	}

	eval := NewEvaluator(vars)
	SetEvaluatorContext(eval, "IF", "test/module")
	SetEvaluatorNoPlatform(eval, module.NoPlatform)
	eval.Evaluate(ast)

	if len(logger.evalVarTraces) != 1 {
		t.Fatalf("Got %d traces, want 1", len(logger.evalVarTraces))
	}

	trace := logger.evalVarTraces[0]
	if trace.NoPlatformValue != true {
		t.Errorf("NoPlatformValue = %v, want true", trace.NoPlatformValue)
	}
}
