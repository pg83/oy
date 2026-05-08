package main

// ResolveConditionals evaluates all conditional blocks in a module, merging
// the appropriate branch (IF, ELSEIF, or ELSE) into the result module based on
// the provided build context and variable set. Each conditional block is
// processed independently: IF condition is checked first, then ELSEIFs in order,
// finally ELSE if present. The matching branch's Dependencies, Sources,
// Properties, and Recursions are merged into the result.
func ResolveConditionals(module *Module, ctx *BuildContext, vars VariableSet) *Module {
	if module == nil {
		return nil
	}

	if len(module.Conditionals) == 0 {
		return module
	}

	resultModule := &Module{
		Type:         module.Type,
		SourcePath:   module.SourcePath,
		Dependencies: append([]string{}, module.Dependencies...),
		Sources:      append([]string{}, module.Sources...),
		Recursions:   append([]*RecurseDirective{}, module.Recursions...),
		Properties:   make(map[string]string),
	}

	for k, v := range module.Properties {
		resultModule.Properties[k] = v
	}

	for _, block := range module.Conditionals {
		var selectedModule *Module

		evaluator := NewEvaluator(vars)

		conditionAST := ParseConditionExpression(block.IfBranch.Condition)
		if evaluator.Evaluate(conditionAST) {
			selectedModule = block.IfBranch.Module

		} else {
			selected := false

			for _, elseifBranch := range block.ElseIfs {
				elseifAST := ParseConditionExpression(elseifBranch.Condition)
				if evaluator.Evaluate(elseifAST) {
					selectedModule = elseifBranch.Module
					selected = true
					break
				}
			}

			if !selected && block.ElseBranch != nil {
				selectedModule = block.ElseBranch.Module
			}
		}

		if selectedModule != nil {
			MergeModule(resultModule, selectedModule)
		}
	}

	return resultModule
}

// MergeModule merges fields from source into target. Dependencies, Sources, and
// Recursions are appended. Properties are copied, with source values overwriting
// target values for matching keys.
func MergeModule(target, source *Module) {
	if target == nil || source == nil {
		return
	}

	for _, dep := range source.Dependencies {
		found := false
		for _, existing := range target.Dependencies {
			if existing == dep {
				found = true
				break
			}
		}

		if !found {
			target.Dependencies = append(target.Dependencies, dep)
		}
	}

	for _, src := range source.Sources {
		found := false
		for _, existing := range target.Sources {
			if existing == src {
				found = true
				break
			}
		}

		if !found {
			target.Sources = append(target.Sources, src)
		}
	}

	if target.Properties == nil {
		target.Properties = make(map[string]string)
	}

	for k, v := range source.Properties {
		target.Properties[k] = v
	}

	for _, rec := range source.Recursions {
		found := false
		for _, existing := range target.Recursions {
			if sameRecurseDirective(existing, rec) {
				found = true
				break
			}
		}

		if !found {
			target.Recursions = append(target.Recursions, rec)
		}
	}
}

func sameRecurseDirective(a, b *RecurseDirective) bool {
	if len(a.Paths) != len(b.Paths) {
		return false
	}

	for i := range a.Paths {
		if a.Paths[i] != b.Paths[i] {
			return false
		}
	}

	return true
}

// EvaluateBuildCondition determines whether a module should be included in the
// build graph based on its BUILD_ONLY_IF directive. If the module has a non-nil
// BuildCondition with a non-empty Expression, the expression is parsed and evaluated.
// BUILD_ONLY_IF semantics: "build only if condition is false", meaning the function
// returns false (discard) when the condition evaluates to true, and true (allow) when
// the condition evaluates to false. Modules without BuildCondition are always allowed.
func EvaluateBuildCondition(module *Module, ctx *BuildContext, vars VariableSet) bool {
	if module == nil || module.BuildCondition == nil {
		return true
	}

	if module.BuildCondition.Expression == "" {
		return true
	}

	evaluator := NewEvaluator(vars)
	conditionAST := ParseConditionExpression(module.BuildCondition.Expression)

	conditionResult := evaluator.Evaluate(conditionAST)

	return !conditionResult
}
