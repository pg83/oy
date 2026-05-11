package main

// ExprOp represents the type of operation in a conditional expression AST.
import (
	"fmt"
	"strconv"
	"strings"
)

type ExprOp int

const (
	ExprIdent ExprOp = iota
	ExprAnd
	ExprOr
	ExprNot
	ExprLessThan
	ExprGreaterThan
	ExprEqual
	ExprNumber
)

// ExpressionAST represents an abstract syntax tree node for conditional expressions
// used in IF/ELSEIF/BUILD_ONLY_IF statements. Supports identifiers (e.g., OS_LINUX),
// logical AND with &&, logical OR with ||, logical NOT with ! operators,
// comparison operators (<, >, ==), and numeric literals.
// The parser (not yet implemented) populates these structures from condition strings.
type ExpressionAST struct {
	Op         ExprOp
	Args       []*ExpressionAST
	Ident      string
	Number     float64
	SourceText string
}

// Evaluator evaluates ExpressionAST nodes against a VariableSet, resolving
// identifiers to boolean values and applying logical operators.
type Evaluator struct {
	vars        VariableSet
	exprContext string
	modulePath  string
	condContext string
	noPlatform  bool
}

// NewEvaluator creates an Evaluator bound to the provided VariableSet.
func NewEvaluator(vars VariableSet) *Evaluator {
	return &Evaluator{
		vars: vars,
	}
}

// NewEvaluatorWithContext creates an Evaluator with context for diagnostic logging.
func SetEvaluatorContext(eval *Evaluator, context, modulePath string) {
	eval.condContext = context
	eval.modulePath = modulePath
}

func SetEvaluatorNoPlatform(eval *Evaluator, noPlatform bool) {
	eval.noPlatform = noPlatform
}

// Evaluate computes the boolean value of an expression AST. For ExprIdent,
// returns vars.IsTrue(ident). For ExprAnd, returns true only if all args are true
// and at least one arg exists. For ExprOr, returns true if any arg is true.
// For ExprNot, returns the negation of the single child argument.
// For ExprLessThan, ExprGreaterThan, ExprEqual, evaluates numeric comparisons.
// For ExprNumber, returns true (literal value stored in Number field for comparisons).
// Panics if args are invalid for the operation type.
func (e *Evaluator) Evaluate(expr *ExpressionAST) bool {
	if expr == nil {
		return false
	}

	switch expr.Op {
	case ExprIdent:
		varIdent := expr.Ident
		if strings.HasPrefix(varIdent, "$") {
			varIdent = varIdent[1:]
		}
		varValue := ""
		if val, hasVal := e.vars.GetValue(varIdent); hasVal {
			varValue = val
		}
		result := e.vars.IsTrue(varIdent)
		if tl := GetGlobalTraversalLogger(); tl != nil && tl.IsEvalTracingEnabled() {
			archValue := e.vars.GetArchValue()
			muslValue := e.vars.GetMuslValue()
			osValue := e.vars.GetOSValue()
			tl.LogEvalVar(varIdent, varValue, e.condContext, e.modulePath, expr.SourceText, result, archValue, muslValue, osValue, e.noPlatform)
		}
		return result

	case ExprAnd:
		if len(expr.Args) == 0 {
			ThrowFmt("invalid expression: AND node missing args")
		}

		for _, arg := range expr.Args {
			if !e.Evaluate(arg) {
				return false
			}
		}

		return true

	case ExprOr:
		if len(expr.Args) == 0 {
			ThrowFmt("invalid expression: OR node missing args")
		}

		for _, arg := range expr.Args {
			if e.Evaluate(arg) {
				return true
			}
		}

		return false

	case ExprNot:
		if len(expr.Args) != 1 {
			ThrowFmt("invalid expression: NOT with %d args (expected 1)", len(expr.Args))
		}

		return !e.Evaluate(expr.Args[0])

	case ExprLessThan:
		if len(expr.Args) != 2 {
			ThrowFmt("invalid expression: < with %d args (expected 2)", len(expr.Args))
		}
		return e.evalComparison(expr.Args[0], expr.Args[1], "<")

	case ExprGreaterThan:
		if len(expr.Args) != 2 {
			ThrowFmt("invalid expression: > with %d args (expected 2)", len(expr.Args))
		}
		return e.evalComparison(expr.Args[0], expr.Args[1], ">")

	case ExprEqual:
		if len(expr.Args) != 2 {
			ThrowFmt("invalid expression: == with %d args (expected 2)", len(expr.Args))
		}
		return e.evalComparison(expr.Args[0], expr.Args[1], "==")

	case ExprNumber:
		return true

	default:
		ThrowFmt("invalid expression: unknown operator %d", expr.Op)
		return false
	}
}

func (e *Evaluator) evalComparison(left, right *ExpressionAST, op string) bool {
	leftNum, leftCanNumeric := e.tryEvalCanCastToNumber(left)
	rightNum, rightCanNumeric := e.tryEvalCanCastToNumber(right)

	if leftCanNumeric && rightCanNumeric {
		switch op {
		case "<":
			return leftNum < rightNum
		case ">":
			return leftNum > rightNum
		case "==":
			return leftNum == rightNum
		default:
			ThrowFmt("invalid comparison operator: %s", op)
			return false
		}
	}

	leftVal, _ := e.evalIdentOrNumberAsString(left)
	rightVal, _ := e.evalIdentOrNumberAsString(right)

	switch op {
	case "<":
		return leftVal < rightVal
	case ">":
		return leftVal > rightVal
	case "==":
		return leftVal == rightVal
	default:
		ThrowFmt("invalid comparison operator: %s", op)
		return false
	}
}

func (e *Evaluator) tryEvalCanCastToNumber(expr *ExpressionAST) (float64, bool) {
	switch expr.Op {
	case ExprNumber:
		return expr.Number, true
	case ExprIdent:
		varIdent := expr.Ident
		if strings.HasPrefix(varIdent, "$") {
			varIdent = varIdent[1:]
		}
		val, ok := e.vars.GetValue(varIdent)
		varValue := ""
		if ok {
			varValue = val
		}
		result := e.vars.IsTrue(varIdent)
		if tl := GetGlobalTraversalLogger(); tl != nil && tl.IsEvalTracingEnabled() {
			archValue := e.vars.GetArchValue()
			muslValue := e.vars.GetMuslValue()
			osValue := e.vars.GetOSValue()
			tl.LogEvalVar(varIdent, varValue, e.condContext, e.modulePath, expr.SourceText, result, archValue, muslValue, osValue, e.noPlatform)
		}
		if !ok {
			return 0, true
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func (e *Evaluator) tryEvalAsNumber(expr *ExpressionAST) (float64, bool) {
	switch expr.Op {
	case ExprNumber:
		return expr.Number, true
	case ExprIdent:
		varIdent := expr.Ident
		if strings.HasPrefix(varIdent, "$") {
			varIdent = varIdent[1:]
		}
		val, ok := e.vars.GetValue(varIdent)
		if !ok {
			return 0, false
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func (e *Evaluator) evalIdentOrNumberAsString(expr *ExpressionAST) (string, bool) {
	switch expr.Op {
	case ExprNumber:
		return fmt.Sprintf("%v", expr.Number), false
	case ExprIdent:
		varIdent := expr.Ident
		if strings.HasPrefix(varIdent, "$") {
			varIdent = varIdent[1:]
		}
		if val, ok := e.vars.GetValue(varIdent); ok {
			return val, true
		}
		return varIdent, true
	default:
		return "", false
	}
}

func (e *Evaluator) evalNumeric(expr *ExpressionAST) float64 {
	switch expr.Op {
	case ExprNumber:
		return expr.Number
	case ExprIdent:
		varIdent := expr.Ident
		if strings.HasPrefix(varIdent, "$") {
			varIdent = varIdent[1:]
		}
		val, ok := e.vars.GetValue(varIdent)
		if !ok {
			return 0
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			ThrowFmt("invalid numeric value for identifier '%s': %s", expr.Ident, val)
		}
		return f
	default:
		ThrowFmt("invalid comparison operand: operator %d", expr.Op)
		return 0
	}
}
