package main

// SourceLocation represents a position in source code for error reporting and debugging.
type SourceLocation struct {
	File   string
	Line   int32
	Column int32
}

// ModuleType represents the type of a ya.make module definition.
type ModuleType int

const (
	ModuleTypeUnknown     ModuleType = iota
	ModuleTypeProgram                // PROGRAM() - executable binary
	ModuleTypeLibrary                // LIBRARY() - static/shared library
	ModuleTypeGoLibrary              // GO_LIBRARY() - Go library
	ModuleTypeDLL                    // DLL() - Windows shared library
	ModuleTypePy23Library            // PY23_LIBRARY() - Python 3.x library
	ModuleTypePyLibrary              // PY_LIBRARY() - Python library (any version)
)

// String returns a human-readable representation of the ModuleType.
func (mt ModuleType) String() string {
	switch mt {
	case ModuleTypeProgram:
		return "PROGRAM"
	case ModuleTypeLibrary:
		return "LIBRARY"
	case ModuleTypeGoLibrary:
		return "GO_LIBRARY"
	case ModuleTypeDLL:
		return "DLL"
	case ModuleTypePy23Library:
		return "PY23_LIBRARY"
	case ModuleTypePyLibrary:
		return "PY_LIBRARY"
	default:
		return "UNKNOWN"
	}
}

// ConditionalBranch represents a single condition-branch pair in a conditional block.
type ConditionalBranch struct {
	Condition string
	Module    *Module
}

// File represents a single ya.make file containing module definitions and recurse directives.
type File struct {
	Modules []*Module
	Imports []*RecurseDirective
}

// Module represents a complete ya.make module definition with all its attributes.
type Module struct {
	Type           ModuleType
	SourcePath     string
	Dependencies   []string
	Sources        []string
	Properties     map[string]string
	Conditionals   []*ConditionalBlock
	Recursions     []*RecurseDirective
	BuildCondition *BuildCondition
}

// RecurseDirective represents a RECURSE or RECURSE_FOR_TESTS directive for including subdirectories.
type RecurseDirective struct {
	Paths    []string
	Location SourceLocation
}

// ConditionalBlock represents an IF/ELSEIF/ELSE/ENDIF conditional block in ya.make.
type ConditionalBlock struct {
	IfBranch   *ConditionalBranch
	ElseIfs    []*ConditionalBranch
	ElseBranch *ConditionalBranch
	Location   SourceLocation
}

// Property represents a SET key-value assignment in ya.make.
type Property struct {
	Key      string
	Value    string
	Location SourceLocation
}

// SourceFileList represents a source file list (SRCS, PY_SRCS, etc.) with language information.
type SourceFileList struct {
	Files    []string
	Language string
}

// BuildCondition represents a BUILD_ONLY_IF directive that conditions module existence.
type BuildCondition struct {
	Expression string
}

// VersionDeclaration represents a VERSION() directive declaring module version.
type VersionDeclaration struct {
	Version string
}

// LicenseDeclaration represents a LICENSE() directive declaring module license.
type LicenseDeclaration struct {
	LicenseType string
	Location    SourceLocation
}

// HasProperty checks if a property with the given key exists in the module.
func (m *Module) HasProperty(key string) bool {
	if m == nil || m.Properties == nil {
		return false
	}

	_, exists := m.Properties[key]
	return exists
}

// GetProperty retrieves the value of a property, returning (value, true) if found.
func (m *Module) GetProperty(key string) (string, bool) {
	if m == nil || m.Properties == nil {
		return "", false
	}

	val, exists := m.Properties[key]
	return val, exists
}

// AddDependency adds a dependency path to the module.
func (m *Module) AddDependency(path string) {
	if m == nil {
		return
	}

	if m.Dependencies == nil {
		m.Dependencies = make([]string, 0, 1)
	}

	m.Dependencies = append(m.Dependencies, path)
}

// AddSource adds a source file path to the module.
func (m *Module) AddSource(path string) {
	if m == nil {
		return
	}

	if m.Sources == nil {
		m.Sources = make([]string, 0, 1)
	}

	m.Sources = append(m.Sources, path)
}

// AddProperty adds or updates a property key-value pair in the module.
func (m *Module) AddProperty(key, value string) {
	if m == nil {
		return
	}

	if m.Properties == nil {
		m.Properties = make(map[string]string)
	}

	m.Properties[key] = value
}

// AddRecurse adds a recurse directive to the module.
func (m *Module) AddRecurse(dir *RecurseDirective) {
	if m == nil {
		return
	}

	if m.Recursions == nil {
		m.Recursions = make([]*RecurseDirective, 0, 1)
	}

	m.Recursions = append(m.Recursions, dir)
}

// AddConditional adds a conditional block to the module.
func (m *Module) AddConditional(cond *ConditionalBlock) {
	if m == nil {
		return
	}

	if m.Conditionals == nil {
		m.Conditionals = make([]*ConditionalBlock, 0, 1)
	}

	m.Conditionals = append(m.Conditionals, cond)
}
