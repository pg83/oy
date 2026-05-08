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
		t.Errorf("expected ExprOr (AND has higher precedence), got %v", expr.Op)
	}

	if expr.Args[0].Op != ExprAnd {
		t.Errorf("expected first arg to be ExprAnd, got %v", expr.Args[0].Op)
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
