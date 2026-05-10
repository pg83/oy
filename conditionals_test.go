package main

import (
	"runtime"
	"testing"
)

func TestBuildContextDefaults(t *testing.T) {
	ctx := NewBuildContext()

	if runtime.GOOS == "linux" {
		if ctx.Platform != PlatformLinux {
			t.Errorf("Expected PlatformLinux, got %v", ctx.Platform)
		}
	} else if runtime.GOOS == "darwin" {
		if ctx.Platform != PlatformDarwin {
			t.Errorf("Expected PlatformDarwin, got %v", ctx.Platform)
		}
	} else if runtime.GOOS == "windows" {
		if ctx.Platform != PlatformWindows {
			t.Errorf("Expected PlatformWindows, got %v", ctx.Platform)
		}
	}

	if runtime.GOARCH == "386" {
		if ctx.Arch != Arch32 {
			t.Errorf("Expected Arch32, got %v", ctx.Arch)
		}
	} else {
		if ctx.Arch != Arch64 {
			t.Errorf("Expected Arch64, got %v", ctx.Arch)
		}
	}

	if ctx.Platform == PlatformDarwin {
		if ctx.Compiler != CompilerClang {
			t.Errorf("Expected CompilerClang, got %v", ctx.Compiler)
		}
	} else if ctx.Platform == PlatformLinux {
		if ctx.Compiler != CompilerGCC {
			t.Errorf("Expected CompilerGCC, got %v", ctx.Compiler)
		}
	}
}

func TestVariableSetIsTrue(t *testing.T) {
	vars := NewMemoryVariableSet()

	trueValues := []string{"yes", "true", "on", "1", "Yes", "YES", "TrUe", "  yes  ", "anything-else"}
	for _, val := range trueValues {
		vars.SetValue("test", val)
		if !vars.IsTrue("test") {
			t.Errorf("Expected true for value %q", val)
		}
	}

	falseValues := []string{"no", "false", "off", "0", "No", "FALSE", "OFF", "  no  ", ""}
	for _, val := range falseValues {
		vars.SetValue("test", val)
		if vars.IsTrue("test") {
			t.Errorf("Expected false for value %q", val)
		}
	}

	if vars.IsTrue("nonexistent") {
		t.Error("Expected false for nonexistent key")
	}
}

func TestBuildContextVariableSetOS(t *testing.T) {
	ctxLinux := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	inner := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(ctxLinux, inner)

	if !vars.IsTrue("OS_LINUX") {
		t.Error("Expected OS_LINUX to be true on Linux")
	}

	if vars.IsTrue("OS_WINDOWS") {
		t.Error("Expected OS_WINDOWS to be false on Linux")
	}

	if vars.IsTrue("OS_DARWIN") {
		t.Error("Expected OS_DARWIN to be false on Linux")
	}

	if vars.IsTrue("OS_MAC") {
		t.Error("Expected OS_MAC (legacy) to be false on Linux")
	}

	ctxDarwin := &BuildContext{Platform: PlatformDarwin, Arch: Arch64, Compiler: CompilerClang}
	varsDarwin := NewBuildContextVariableSet(ctxDarwin, NewMemoryVariableSet())
	if varsDarwin.IsTrue("OS_LINUX") {
		t.Error("Expected OS_LINUX to be false on Darwin")
	}
	if !varsDarwin.IsTrue("OS_DARWIN") {
		t.Error("Expected OS_DARWIN to be true on Darwin")
	}
}

func TestBuildContextVariableSetArch(t *testing.T) {
	ctx64 := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewBuildContextVariableSet(ctx64, NewMemoryVariableSet())

	if !vars.IsTrue("ARCH_TYPE_64") {
		t.Error("Expected ARCH_TYPE_64 to be true on 64-bit")
	}

	if vars.IsTrue("ARCH_TYPE_32") {
		t.Error("Expected ARCH_TYPE_32 to be false on 64-bit")
	}

	ctx32 := &BuildContext{Platform: PlatformLinux, Arch: Arch32, Compiler: CompilerGCC}
	vars32 := NewBuildContextVariableSet(ctx32, NewMemoryVariableSet())
	if vars32.IsTrue("ARCH_TYPE_64") {
		t.Error("Expected ARCH_TYPE_64 to be false on 32-bit")
	}
	if !vars32.IsTrue("ARCH_TYPE_32") {
		t.Error("Expected ARCH_TYPE_32 to be true on 32-bit")
	}
}

func TestBuildContextVariableSetCompiler(t *testing.T) {
	ctxGCC := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewBuildContextVariableSet(ctxGCC, NewMemoryVariableSet())

	if !vars.IsTrue("GCC") {
		t.Error("Expected GCC to be true")
	}

	if vars.IsTrue("CLANG") {
		t.Error("Expected CLANG to be false")
	}

	if vars.IsTrue("MSVC") {
		t.Error("Expected MSVC to be false")
	}

	ctxClang := &BuildContext{Platform: PlatformDarwin, Arch: Arch64, Compiler: CompilerClang}
	varsClang := NewBuildContextVariableSet(ctxClang, NewMemoryVariableSet())
	if !varsClang.IsTrue("CLANG") {
		t.Error("Expected CLANG to be true")
	}
}

func TestEvaluatorIdent(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "yes")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op:    ExprIdent,
		Ident: "OS_LINUX",
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for OS_LINUX=yes")
	}
}

func TestEvaluatorNot(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "yes")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprNot,
		Args: []*ExpressionAST{
			{
				Op:    ExprIdent,
				Ident: "OS_LINUX",
			},
		},
	}

	if evaluator.Evaluate(expr) {
		t.Error("Expected false for NOT OS_LINUX=true")
	}
}

func TestEvaluatorAnd(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("A", "yes")
	vars.SetValue("B", "yes")
	vars.SetValue("C", "no")

	evaluator := NewEvaluator(vars)

	trueExpr := &ExpressionAST{
		Op: ExprAnd,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "A"},
			{Op: ExprIdent, Ident: "B"},
		},
	}
	if !evaluator.Evaluate(trueExpr) {
		t.Error("Expected true for A && B (both true)")
	}

	falseExpr := &ExpressionAST{
		Op: ExprAnd,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "A"},
			{Op: ExprIdent, Ident: "C"},
		},
	}
	if evaluator.Evaluate(falseExpr) {
		t.Error("Expected false for A && C (C is false)")
	}
}

func TestEvaluatorOr(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("A", "yes")
	vars.SetValue("B", "no")

	evaluator := NewEvaluator(vars)

	trueExpr := &ExpressionAST{
		Op: ExprOr,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "A"},
			{Op: ExprIdent, Ident: "B"},
		},
	}
	if !evaluator.Evaluate(trueExpr) {
		t.Error("Expected true for A || B (A is true)")
	}

	falseExpr := &ExpressionAST{
		Op: ExprOr,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "B"},
			{Op: ExprIdent, Ident: "C"},
		},
	}
	if evaluator.Evaluate(falseExpr) {
		t.Error("Expected false for B || C (both false/missing)")
	}
}

func TestEvaluatorNil(t *testing.T) {
	vars := NewMemoryVariableSet()
	evaluator := NewEvaluator(vars)

	if evaluator.Evaluate(nil) {
		t.Error("Expected false for nil expression")
	}
}

func TestResolveConditionalsIF(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "yes")

	baseModule := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "lib/base",
		Dependencies: []string{"base_dep"},
		Sources:      []string{"base.cpp"},
		Properties:   map[string]string{"base_key": "base_val"},
		Conditionals: []*ConditionalBlock{
			{
				IfBranch: &ConditionalBranch{
					Condition: "OS_LINUX",
					Module: &Module{
						Dependencies: []string{"linux_dep"},
						Sources:      []string{"linux.cpp"},
						Properties:   map[string]string{"linux_key": "linux_val"},
					},
				},
			},
		},
	}

	result := ResolveConditionals(baseModule, ctx, vars)

	if len(result.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(result.Dependencies))
	}

	if len(result.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(result.Sources))
	}

	if result.Properties["linux_key"] != "linux_val" {
		t.Error("Expected linux_key property to be merged")
	}
}

func TestResolveConditionalsELSEIF(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformWindows, Arch: Arch64, Compiler: CompilerMSVC}
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_WINDOWS", "yes")
	vars.SetValue("OS_LINUX", "no")

	baseModule := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "lib/base",
		Conditionals: []*ConditionalBlock{
			{
				IfBranch: &ConditionalBranch{
					Condition: "OS_LINUX",
					Module: &Module{
						Sources: []string{"linux.cpp"},
					},
				},
				ElseIfs: []*ConditionalBranch{
					{
						Condition: "OS_WINDOWS",
						Module: &Module{
							Sources: []string{"windows.cpp"},
						},
					},
				},
			},
		},
	}

	result := ResolveConditionals(baseModule, ctx, vars)

	if len(result.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(result.Sources))
	}

	if result.Sources[0] != "windows.cpp" {
		t.Errorf("Expected windows.cpp, got %s", result.Sources[0])
	}
}

func TestResolveConditionalsELSE(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformDarwin, Arch: Arch64, Compiler: CompilerClang}
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "no")
	vars.SetValue("OS_WINDOWS", "no")

	baseModule := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "lib/base",
		Conditionals: []*ConditionalBlock{
			{
				IfBranch: &ConditionalBranch{
					Condition: "OS_LINUX",
					Module: &Module{
						Sources: []string{"linux.cpp"},
					},
				},
				ElseIfs: []*ConditionalBranch{
					{
						Condition: "OS_WINDOWS",
						Module: &Module{
							Sources: []string{"windows.cpp"},
						},
					},
				},
				ElseBranch: &ConditionalBranch{
					Module: &Module{
						Sources: []string{"other.cpp"},
					},
				},
			},
		},
	}

	result := ResolveConditionals(baseModule, ctx, vars)

	if len(result.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(result.Sources))
	}

	if result.Sources[0] != "other.cpp" {
		t.Errorf("Expected other.cpp, got %s", result.Sources[0])
	}
}

func TestResolveConditionalsNoMatch(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformDarwin, Arch: Arch64, Compiler: CompilerClang}
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "no")
	vars.SetValue("OS_WINDOWS", "no")

	baseModule := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "lib/base",
		Sources:    []string{"base.cpp"},
		Conditionals: []*ConditionalBlock{
			{
				IfBranch: &ConditionalBranch{
					Condition: "OS_LINUX",
					Module: &Module{
						Sources: []string{"linux.cpp"},
					},
				},
				ElseIfs: []*ConditionalBranch{
					{
						Condition: "OS_WINDOWS",
						Module: &Module{
							Sources: []string{"windows.cpp"},
						},
					},
				},
			},
		},
	}

	result := ResolveConditionals(baseModule, ctx, vars)

	if len(result.Sources) != 1 {
		t.Errorf("Expected 1 source (base only), got %d", len(result.Sources))
	}

	if result.Sources[0] != "base.cpp" {
		t.Errorf("Expected base.cpp, got %s", result.Sources[0])
	}
}

func TestEvaluateBuildConditionNoCondition(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "lib/test",
	}

	if !EvaluateBuildCondition(module, ctx, vars) {
		t.Error("Expected true (allow build) for module without BuildCondition")
	}
}

func TestEvaluateBuildConditionNilCondition(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()

	module := &Module{
		Type:           ModuleTypeLibrary,
		SourcePath:     "lib/test",
		BuildCondition: nil,
	}

	if !EvaluateBuildCondition(module, ctx, vars) {
		t.Error("Expected true (allow build) for module with nil BuildCondition")
	}
}

func TestEvaluateBuildConditionTrue(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "yes")

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "lib/test",
		BuildCondition: &BuildCondition{
			Expression: "OS_LINUX",
		},
	}

	if EvaluateBuildCondition(module, ctx, vars) {
		t.Error("Expected false (discard) for BUILD_ONLY_IF with true condition")
	}
}

func TestEvaluateBuildConditionFalse(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_LINUX", "yes")

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "lib/test",
		BuildCondition: &BuildCondition{
			Expression: "OS_WINDOWS",
		},
	}

	if !EvaluateBuildCondition(module, ctx, vars) {
		t.Error("Expected true (allow build) for BUILD_ONLY_IF with false condition")
	}
}

func TestEvaluateBuildConditionEmptyExpression(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()

	module := &Module{
		Type:       ModuleTypeLibrary,
		SourcePath: "lib/test",
		BuildCondition: &BuildCondition{
			Expression: "",
		},
	}

	if !EvaluateBuildCondition(module, ctx, vars) {
		t.Error("Expected true (allow build) for BUILD_ONLY_IF with empty expression")
	}
}

func TestEvaluateBuildConditionNilModule(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()

	if !EvaluateBuildCondition(nil, ctx, vars) {
		t.Error("Expected true (allow build) for nil module")
	}
}

func TestMergeModule(t *testing.T) {
	target := &Module{
		Dependencies: []string{"a", "b"},
		Sources:      []string{"a.cpp", "b.cpp"},
		Properties:   map[string]string{"key1": "val1"},
		Recursions:   []*RecurseDirective{{Paths: []string{"recurse1"}}},
	}

	source := &Module{
		Dependencies: []string{"b", "c"},
		Sources:      []string{"b.cpp", "c.cpp"},
		Properties:   map[string]string{"key1": "new_val", "key2": "val2"},
		Recursions:   []*RecurseDirective{{Paths: []string{"recurse1"}}, {Paths: []string{"recurse2"}}},
	}

	MergeModule(target, source)

	if len(target.Dependencies) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(target.Dependencies))
	}

	if len(target.Sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(target.Sources))
	}

	if target.Properties["key1"] != "new_val" {
		t.Error("Expected key1 to be overwritten")
	}

	if target.Properties["key2"] != "val2" {
		t.Error("Expected key2 to be added")
	}

	if len(target.Recursions) != 2 {
		t.Errorf("Expected 2 recursions, got %d", len(target.Recursions))
	}
}

func TestEvaluatorComparisonLessThan(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("ANDROID_API", "29")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprLessThan,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "ANDROID_API"},
			{Op: ExprNumber, Number: 30},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for 29 < 30")
	}
}

func TestEvaluatorComparisonGreaterThan(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("VERSION", "3")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprGreaterThan,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "VERSION"},
			{Op: ExprNumber, Number: 2},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for 3 > 2")
	}
}

func TestEvaluatorComparisonEqual(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("LEVEL", "5")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprEqual,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "LEVEL"},
			{Op: ExprNumber, Number: 5},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for 5 == 5")
	}
}

func TestEvaluatorComparisonLessThanFalse(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("ANDROID_API", "30")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprLessThan,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "ANDROID_API"},
			{Op: ExprNumber, Number: 29},
		},
	}

	if evaluator.Evaluate(expr) {
		t.Error("Expected false for 30 < 29")
	}
}

func TestEvaluatorComparisonEqualFalse(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("LEVEL", "6")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprEqual,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "LEVEL"},
			{Op: ExprNumber, Number: 5},
		},
	}

	if evaluator.Evaluate(expr) {
		t.Error("Expected false for 6 == 5")
	}
}

func TestEvaluatorComparisonMissingVariable(t *testing.T) {
	vars := NewMemoryVariableSet()

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprLessThan,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "MISSING"},
			{Op: ExprNumber, Number: 30},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for 0 < 30 (missing variable defaults to 0)")
	}
}

func TestEvaluatorComparisonNumberDecimal(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("VERSION", "3.14")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprLessThan,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "VERSION"},
			{Op: ExprNumber, Number: 3.15},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for 3.14 < 3.15")
	}
}

func TestEvaluatorComparisonBooleanAndComparison(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("OS_ANDROID", "yes")
	vars.SetValue("ANDROID_API", "29")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprAnd,
		Args: []*ExpressionAST{
			{Op: ExprIdent, Ident: "OS_ANDROID"},
			{
				Op: ExprLessThan,
				Args: []*ExpressionAST{
					{Op: ExprIdent, Ident: "ANDROID_API"},
					{Op: ExprNumber, Number: 30},
				},
			},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for OS_ANDROID && ANDROID_API < 30")
	}
}

func TestEvaluatorComparisonBooleanOrComparison(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("API1", "29")
	vars.SetValue("API2", "30")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprOr,
		Args: []*ExpressionAST{
			{
				Op: ExprEqual,
				Args: []*ExpressionAST{
					{Op: ExprIdent, Ident: "API1"},
					{Op: ExprNumber, Number: 29},
				},
			},
			{
				Op: ExprEqual,
				Args: []*ExpressionAST{
					{Op: ExprIdent, Ident: "API2"},
					{Op: ExprNumber, Number: 30},
				},
			},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for API1 == 29 OR API2 == 30")
	}
}

func TestEvaluatorComparisonWithNot(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("VERSION", "2")

	evaluator := NewEvaluator(vars)
	expr := &ExpressionAST{
		Op: ExprNot,
		Args: []*ExpressionAST{
			{
				Op: ExprEqual,
				Args: []*ExpressionAST{
					{Op: ExprIdent, Ident: "VERSION"},
					{Op: ExprNumber, Number: 1},
				},
			},
		},
	}

	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for NOT VERSION == 1")
	}
}

func TestParseAndEvaluateComparison(t *testing.T) {
	vars := NewMemoryVariableSet()
	vars.SetValue("ANDROID_API", "29")

	expr := ParseConditionExpression("ANDROID_API < 30")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	evaluator := NewEvaluator(vars)
	if !evaluator.Evaluate(expr) {
		t.Error("Expected true for ANDROID_API < 30")
	}
}

func TestResolveConditionalsWithComparison(t *testing.T) {
	ctx := &BuildContext{Platform: PlatformLinux, Arch: Arch64, Compiler: CompilerGCC}
	vars := NewMemoryVariableSet()
	vars.SetValue("ANDROID_API", "29")

	baseModule := &Module{
		Type:         ModuleTypeLibrary,
		SourcePath:   "lib/base",
		Dependencies: []string{"base_dep"},
		Sources:      []string{"base.cpp"},
		Properties:   map[string]string{"base_key": "base_val"},
		Conditionals: []*ConditionalBlock{
			{
				IfBranch: &ConditionalBranch{
					Condition: "ANDROID_API < 30",
					Module: &Module{
						Dependencies: []string{"old_android_dep"},
						Sources:      []string{"old_android.cpp"},
						Properties:   map[string]string{"old_key": "old_val"},
					},
				},
			},
		},
	}

	result := ResolveConditionals(baseModule, ctx, vars)

	if len(result.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(result.Dependencies))
	}

	if result.Properties["old_key"] != "old_val" {
		t.Error("Expected old_key property to be merged")
	}
}
