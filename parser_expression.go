package main

// ParseConditionExpression is a stub that converts a condition string into an
// ExpressionAST. The full parser implementation (which correctly handles
// operators like NOT, AND, OR) will be implemented in a future ticket. For now,
// this stub treats the entire string as a single identifier.
func ParseConditionExpression(conditionExpr string) *ExpressionAST {
	if conditionExpr == "" {
		return nil
	}

	return &ExpressionAST{
		Op:    ExprIdent,
		Ident: conditionExpr,
	}
}
