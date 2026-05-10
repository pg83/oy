package main

import (
	"path/filepath"
	"strings"
)

type TransitiveResolver struct {
	registry *ModuleRegistry
}

func NewTransitiveResolver(registry *ModuleRegistry) *TransitiveResolver {
	return &TransitiveResolver{
		registry: registry,
	}
}

func (t *TransitiveResolver) ComputeTransitiveDeps(module *Module) map[string]struct{} {
	if module == nil {
		return make(map[string]struct{})
	}

	result := make(map[string]struct{})
	visited := make(map[string]bool)

	t.computeTransitiveDepsDFS(module, result, visited)

	return result
}

func (t *TransitiveResolver) computeTransitiveDepsDFS(module *Module, result map[string]struct{}, visited map[string]bool) {
	if module == nil {
		return
	}

	moduleKey := NormalizedPath(module.SourcePath)

	if visited[moduleKey] {
		if _, exists := result[moduleKey]; !exists {
			ThrowFmt("circular dependency detected at module: %s", module.SourcePath)
		}
		return
	}

	visited[moduleKey] = true

	for _, depPath := range module.Dependencies {
		normalizedPath := t.normalizeDepPath(module.SourcePath, depPath)

		depModule := t.registry.Get(normalizedPath)
		if depModule == nil {
			continue
		}

		uid := NewUID([]byte(depModule.SourcePath))

		if _, exists := result[uid]; !exists {
			result[uid] = struct{}{}

			t.computeTransitiveDepsDFS(depModule, result, visited)
		}
	}
}

func (t *TransitiveResolver) normalizeDepPath(modulePath, depPath string) string {
	if filepath.IsAbs(depPath) {
		return NormalizedPath(depPath)
	}

	// Top-level modules like 'util' should not be treated as relative paths
	if depPath == "util" {
		return "util"
	}

	if !strings.HasPrefix(depPath, "library/") &&
		!strings.HasPrefix(depPath, "contrib/") &&
		!strings.HasPrefix(depPath, "yt/") &&
		!strings.HasPrefix(depPath, "arcadia/") &&
		!strings.HasPrefix(depPath, "util/") {
		relDir := filepath.Dir(modulePath)
		absPath := filepath.Join(relDir, depPath)
		return NormalizedPath(absPath)
	}

	return NormalizedPath(depPath)
}
