package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDiagEvalFlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantDiag bool
	}{
		{
			name:     "no diag flag",
			args:     []string{"ymake", "tools/archiver"},
			wantDiag: false,
		},
		{
			name:     "diag eval flag present",
			args:     []string{"ymake", "--diag-eval", "tools/archiver"},
			wantDiag: true,
		},
		{
			name:     "diag eval with other flags",
			args:     []string{"ymake", "--musl", "--diag-eval", "--benchmark", "tools/archiver"},
			wantDiag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}
			if result.DiagEval != tt.wantDiag {
				t.Errorf("ParseFlags().DiagEval = %v, want %v", result.DiagEval, tt.wantDiag)
			}
		})
	}
}

func TestEvalVarLogging(t *testing.T) {
	tests := []struct {
		name         string
		vars         map[string]string
		expr         string
		wantResult   bool
		wantLogCount int
	}{
		{
			name:         "simple identifier true",
			vars:         map[string]string{"OS_LINUX": "true"},
			expr:         "OS_LINUX",
			wantResult:   true,
			wantLogCount: 1,
		},
		{
			name:         "simple identifier false",
			vars:         map[string]string{"MSVC": "false"},
			expr:         "MSVC",
			wantResult:   false,
			wantLogCount: 1,
		},
		{
			name:         "and expression",
			vars:         map[string]string{"OS_LINUX": "true", "MUSL": "yes"},
			expr:         "OS_LINUX && MUSL",
			wantResult:   true,
			wantLogCount: 2,
		},
		{
			name:         "or expression",
			vars:         map[string]string{"MSVC": "false", "CLANG": "true"},
			expr:         "MSVC || CLANG",
			wantResult:   true,
			wantLogCount: 2,
		},
		{
			name:         "not expression",
			vars:         map[string]string{"MSVC": "false"},
			expr:         "!MSVC",
			wantResult:   true,
			wantLogCount: 1,
		},
		{
			name:         "comparison expression",
			vars:         map[string]string{"ANDROID_API": "30"},
			expr:         "ANDROID_API < 31",
			wantResult:   true,
			wantLogCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := NewMemoryVariableSet()
			for k, v := range tt.vars {
				vars.SetValue(k, v)
			}

			logger := NewTraversalLogger(true, true, false)
			SetGlobalTraversalLogger(logger)

			ast := ParseConditionExpression(tt.expr)
			if ast == nil {
				t.Fatalf("ParseConditionExpression(%q) returned nil", tt.expr)
			}

			eval := NewEvaluator(vars)
			result := eval.Evaluate(ast)

			if result != tt.wantResult {
				t.Errorf("Evaluate() = %v, want %v", result, tt.wantResult)
			}

			if len(logger.evalVarTraces) != tt.wantLogCount {
				t.Errorf("Got %d var traces, want %d", len(logger.evalVarTraces), tt.wantLogCount)
			}
		})
	}
}

func TestEvalVarTraceStructure(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("PREBUILT", "no")
	vars.SetValue("MUSL", "yes")

	logger := NewTraversalLogger(true, true, false)
	SetGlobalTraversalLogger(logger)

	module := &Module{
		SourcePath: "tools/test",
	}

	ApplyModuleFlags(module, vars)
	ast := ParseConditionExpression("MUSL")
	if ast == nil {
		t.Fatal("ParseConditionExpression returned nil")
	}

	eval := NewEvaluator(vars)
	result := eval.Evaluate(ast)

	if !result {
		t.Errorf("Expected MUSL to evaluate to true")
	}

	if len(logger.evalVarTraces) != 1 {
		t.Fatalf("Got %d traces, want 1", len(logger.evalVarTraces))
	}

	trace := logger.evalVarTraces[0]
	if trace.VariableName != "MUSL" {
		t.Errorf("VariableName = %s, want MUSL", trace.VariableName)
	}
	if trace.VariableValue != "yes" {
		t.Errorf("VariableValue = %s, want yes", trace.VariableValue)
	}
	if trace.Result != true {
		t.Errorf("Result = %v, want true", trace.Result)
	}
	if trace.Expression != "MUSL" {
		t.Errorf("Expression = %s, want MUSL", trace.Expression)
	}
}

func TestEvalVarOutput(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "true")
	vars.SetValue("PREBUILT", "no")

	logger := NewTraversalLogger(true, true, false)
	SetGlobalTraversalLogger(logger)

	ast1 := ParseConditionExpression("OS_LINUX")
	ast2 := ParseConditionExpression("PREBUILT")

	eval := NewEvaluator(vars)
	eval.Evaluate(ast1)
	eval.Evaluate(ast2)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger.OutputSummary()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	outputStr := buf.String()

	if !strings.Contains(outputStr, "VARIABLE EVALUATION DIAGNOSTIC SUMMARY") {
		t.Errorf("Output missing evaluation summary header")
	}
	if !strings.Contains(outputStr, "OS_LINUX") {
		t.Errorf("Output missing OS_LINUX variable")
	}
	if !strings.Contains(outputStr, "PREBUILT") {
		t.Errorf("Output missing PREBUILT variable")
	}
}

func TestPrebuiltVariableTracing(t *testing.T) {
	tests := []struct {
		name       string
		prebuilt   string
		wantResult bool
	}{
		{
			name:       "PREBUILT=yes",
			prebuilt:   "yes",
			wantResult: true,
		},
		{
			name:       "PREBUILT=no",
			prebuilt:   "no",
			wantResult: false,
		},
		{
			name:       "PREBUILT=auto",
			prebuilt:   "auto",
			wantResult: true,
		},
		{
			name:       "PREBUILT unset",
			prebuilt:   "",
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := NewMemoryVariableSet()
			if tt.prebuilt != "" {
				vars.SetValue("PREBUILT", tt.prebuilt)
			}

			logger := NewTraversalLogger(true, true, false)
			SetGlobalTraversalLogger(logger)

			ast := ParseConditionExpression("PREBUILT")
			eval := NewEvaluator(vars)
			result := eval.Evaluate(ast)

			if result != tt.wantResult {
				t.Errorf("PREBUILT=%q evaluated to %v, want %v", tt.prebuilt, result, tt.wantResult)
			}

			if len(logger.evalVarTraces) != 1 {
				t.Fatalf("Got %d traces, want 1", len(logger.evalVarTraces))
			}

			trace := logger.evalVarTraces[0]
			if trace.VariableName != "PREBUILT" {
				t.Errorf("VariableName = %s, want PREBUILT", trace.VariableName)
			}
		})
	}
}

func TestMuslVariableTracing(t *testing.T) {
	tests := []struct {
		name       string
		musl       string
		wantResult bool
	}{
		{
			name:       "MUSL=yes",
			musl:       "yes",
			wantResult: true,
		},
		{
			name:       "MUSL=no",
			musl:       "no",
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := NewMemoryVariableSet()
			vars.SetValue("MUSL", tt.musl)

			logger := NewTraversalLogger(true, true, false)
			SetGlobalTraversalLogger(logger)

			ast := ParseConditionExpression("MUSL")
			eval := NewEvaluator(vars)
			result := eval.Evaluate(ast)

			if result != tt.wantResult {
				t.Errorf("MUSL=%q evaluated to %v, want %v", tt.musl, result, tt.wantResult)
			}

			if len(logger.evalVarTraces) != 1 {
				t.Fatalf("Got %d traces, want 1", len(logger.evalVarTraces))
			}

			_, ok := vars.GetValue("MUSL")
			if !ok {
				t.Errorf("MUSL variable not found")
			}
		})
	}
}
