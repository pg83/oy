package main

import (
	"testing"
)

func TestParseConditionExpression_SimpleIdent(t *testing.T) {
	expr := ParseConditionExpression("OS_LINUX")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprIdent {
		t.Errorf("expected ExprIdent, got %v", expr.Op)
	}

	if expr.Ident != "OS_LINUX" {
		t.Errorf("expected ident 'OS_LINUX', got '%s'", expr.Ident)
	}
}

func TestParseConditionExpression_Not(t *testing.T) {
	expr := ParseConditionExpression("NOT MSVC")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprNot {
		t.Errorf("expected ExprNot, got %v", expr.Op)
	}

	if len(expr.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "MSVC" {
		t.Errorf("expected ident 'MSVC', got '%s'", expr.Args[0].Ident)
	}
}

func TestParseConditionExpression_And(t *testing.T) {
	expr := ParseConditionExpression("OS_LINUX AND ARCH_X86_64")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprAnd {
		t.Errorf("expected ExprAnd, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "OS_LINUX" {
		t.Errorf("expected ident 'OS_LINUX', got '%s'", expr.Args[0].Ident)
	}

	if expr.Args[1].Ident != "ARCH_X86_64" {
		t.Errorf("expected ident 'ARCH_X86_64', got '%s'", expr.Args[1].Ident)
	}
}

func TestParseConditionExpression_Or(t *testing.T) {
	expr := ParseConditionExpression("OS_WINDOWS OR OS_DARWIN")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "OS_WINDOWS" {
		t.Errorf("expected ident 'OS_WINDOWS', got '%s'", expr.Args[0].Ident)
	}

	if expr.Args[1].Ident != "OS_DARWIN" {
		t.Errorf("expected ident 'OS_DARWIN', got '%s'", expr.Args[1].Ident)
	}
}

func TestParseConditionExpression_ComplexParens(t *testing.T) {
	expr := ParseConditionExpression("(OS_LINUX AND ARCH_64) OR (OS_WINDOWS AND NOT CLANG)")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	left := expr.Args[0]
	if left.Op != ExprAnd {
		t.Errorf("expected left to be ExprAnd, got %v", left.Op)
	}

	right := expr.Args[1]
	if right.Op != ExprAnd {
		t.Errorf("expected right to be ExprAnd, got %v", right.Op)
	}

	if len(right.Args) != 2 {
		t.Fatalf("expected right to have 2 args, got %d", len(right.Args))
	}

	if right.Args[1].Op != ExprNot {
		t.Errorf("expected right.[1] to be ExprNot, got %v", right.Args[1].Op)
	}
}

func TestParseConditionExpression_AndPrecedence(t *testing.T) {
	expr := ParseConditionExpression("A AND B OR C")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr (lower precedence), got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Op != ExprAnd {
		t.Errorf("expected first arg to be ExprAnd (higher precedence), got %v", expr.Args[0].Op)
	}
}

func TestParseConditionExpression_NotPrecedence(t *testing.T) {
	expr := ParseConditionExpression("NOT A AND B")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprAnd {
		t.Errorf("expected ExprAnd (NOT has highest precedence), got %v", expr.Op)
	}

	if expr.Args[0].Op != ExprNot {
		t.Errorf("expected first arg to be ExprNot, got %v", expr.Args[0].Op)
	}
}

func TestParseConditionExpression_DoubleNot(t *testing.T) {
	expr := ParseConditionExpression("NOT NOT A")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprNot {
		t.Errorf("expected ExprNot, got %v", expr.Op)
	}

	if expr.Args[0].Op != ExprNot {
		t.Errorf("expected inner to be ExprNot, got %v", expr.Args[0].Op)
	}
}

func TestParseConditionExpression_AmpersandAmpersand(t *testing.T) {
	expr := ParseConditionExpression("A && B")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprAnd {
		t.Errorf("expected ExprAnd, got %v", expr.Op)
	}
}

func TestParseConditionExpression_PipePipe(t *testing.T) {
	expr := ParseConditionExpression("A || B")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr, got %v", expr.Op)
	}
}

func TestParseConditionExpression_Bang(t *testing.T) {
	expr := ParseConditionExpression("!A")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprNot {
		t.Errorf("expected ExprNot, got %v", expr.Op)
	}
}

func TestParseConditionExpression_Empty(t *testing.T) {
	expr := ParseConditionExpression("")
	if expr != nil {
		t.Error("expected nil expression for empty string")
	}
}

func TestParseConditionExpression_Whitespace(t *testing.T) {
	expr := ParseConditionExpression("   ")
	if expr != nil {
		t.Error("expected nil expression for whitespace-only string")
	}
}

func TestParseConditionExpression_TrailingWhitespace(t *testing.T) {
	expr := ParseConditionExpression("OS_LINUX  ")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Ident != "OS_LINUX" {
		t.Errorf("expected ident 'OS_LINUX', got '%s'", expr.Ident)
	}
}

func TestParseConditionExpression_OrPrecedenceWithKeyword(t *testing.T) {
	expr := ParseConditionExpression("GCC OR CLANG OR CLANG_CL")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	left := expr.Args[0]
	if left.Ident != "GCC" && left.Op != ExprOr {
		t.Errorf("expected left to be 'GCC' or ExprOr, got %v", left)
	}
}

func TestParseConditionExpression_NotKeyword(t *testing.T) {
	expr := ParseConditionExpression("NOT OS_EMSCRIPTEN")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprNot {
		t.Errorf("expected ExprNot, got %v", expr.Op)
	}

	if len(expr.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "OS_EMSCRIPTEN" {
		t.Errorf("expected ident 'OS_EMSCRIPTEN', got '%s'", expr.Args[0].Ident)
	}
}

func TestParseConditionExpression_ArchOrArchComplex(t *testing.T) {
	expr := ParseConditionExpression("ARCH_X86_64 OR ARCH_I386")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "ARCH_X86_64" {
		t.Errorf("expected ident 'ARCH_X86_64', got '%s'", expr.Args[0].Ident)
	}

	if expr.Args[1].Ident != "ARCH_I386" {
		t.Errorf("expected ident 'ARCH_I386', got '%s'", expr.Args[1].Ident)
	}
}

func TestParseConditionExpression_ErrorMissingRParen(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing )")
		}
	}()

	ParseConditionExpression("(OS_LINUX")
}

func TestParseConditionExpression_ErrorMissingLParen(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing (")
		}
	}()

	ParseConditionExpression("OS_LINUX)")
}

func TestParseConditionExpression_ErrorUnexpectedToken(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unexpected token after expression")
		}
	}()

	ParseConditionExpression("OS_LINUX OR")
}

func TestParseConditionExpression_LessThan(t *testing.T) {
	expr := ParseConditionExpression("ANDROID_API < 29")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprLessThan {
		t.Errorf("expected ExprLessThan, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "ANDROID_API" {
		t.Errorf("expected ident 'ANDROID_API', got '%s'", expr.Args[0].Ident)
	}

	if expr.Args[1].Number != 29 {
		t.Errorf("expected number 29, got %f", expr.Args[1].Number)
	}
}

func TestParseConditionExpression_GreaterThan(t *testing.T) {
	expr := ParseConditionExpression("VERSION > 2")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprGreaterThan {
		t.Errorf("expected ExprGreaterThan, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "VERSION" {
		t.Errorf("expected ident 'VERSION', got '%s'", expr.Args[0].Ident)
	}

	if expr.Args[1].Number != 2 {
		t.Errorf("expected number 2, got %f", expr.Args[1].Number)
	}
}

func TestParseConditionExpression_Equals(t *testing.T) {
	expr := ParseConditionExpression("LEVEL == 5")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprEqual {
		t.Errorf("expected ExprEqual, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "LEVEL" {
		t.Errorf("expected ident 'LEVEL', got '%s'", expr.Args[0].Ident)
	}

	if expr.Args[1].Number != 5 {
		t.Errorf("expected number 5, got %f", expr.Args[1].Number)
	}
}

func TestParseConditionExpression_NumberDecimal(t *testing.T) {
	expr := ParseConditionExpression("VERSION > 3.14")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[1].Number != 3.14 {
		t.Errorf("expected number 3.14, got %f", expr.Args[1].Number)
	}
}

func TestParseConditionExpression_ComparisonWithAnd(t *testing.T) {
	expr := ParseConditionExpression("OS_ANDROID AND ANDROID_API < 29")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprAnd {
		t.Errorf("expected ExprAnd, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}

	if expr.Args[0].Ident != "OS_ANDROID" {
		t.Errorf("expected ident 'OS_ANDROID', got '%s'", expr.Args[0].Ident)
	}

	left := expr.Args[1]
	if left.Op != ExprLessThan {
		t.Errorf("expected ExprLessThan, got %v", left.Op)
	}
}

func TestParseConditionExpression_ComparisonWithOr(t *testing.T) {
	expr := ParseConditionExpression("API == 29 OR API == 30")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}
}

func TestParseConditionExpression_ComparisonWithNot(t *testing.T) {
	expr := ParseConditionExpression("NOT VERSION == 1")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprNot {
		t.Errorf("expected ExprNot, got %v", expr.Op)
	}

	if len(expr.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(expr.Args))
	}

	if expr.Args[0].Op != ExprEqual {
		t.Errorf("expected ExprEqual, got %v", expr.Args[0].Op)
	}
}

func TestParseConditionExpression_ComparisonWithParens(t *testing.T) {
	expr := ParseConditionExpression("(OS_ANDROID AND ANDROID_API < 29)")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprAnd {
		t.Errorf("expected ExprAnd, got %v", expr.Op)
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}
}

func TestParseConditionExpression_PrecedenceComparisonAnd(t *testing.T) {
	expr := ParseConditionExpression("A AND B < 10")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprAnd {
		t.Errorf("expected ExprAnd (comparison has higher precedence), got %v", expr.Op)
	}

	if expr.Args[1].Op != ExprLessThan {
		t.Errorf("expected second arg to be ExprLessThan, got %v", expr.Args[1].Op)
	}
}

func TestParseConditionExpression_PrecedenceComparisonOr(t *testing.T) {
	expr := ParseConditionExpression("A == 10 OR B == 20")
	if expr == nil {
		t.Fatal("expected non-nil expression")
	}

	if expr.Op != ExprOr {
		t.Errorf("expected ExprOr (comparison has higher precedence than OR), got %v", expr.Op)
	}
}
