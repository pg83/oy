package main

import (
	"path/filepath"
	"strings"
)

type Resolver struct {
	registry *ModuleRegistry
	ctx      *ParseContext
}

func NewResolver(registry *ModuleRegistry, ctx *ParseContext) *Resolver {
	return &Resolver{
		registry: registry,
		ctx:      ctx,
	}
}

func (r *Resolver) ResolveDependencies(module *Module) ([]string, error) {
	if module == nil {
		return nil, nil
	}

	resolvedUIDs := make([]string, 0, len(module.Dependencies))

	for _, depPath := range module.Dependencies {
		normalizedPath := r.normalizeDepPath(module.SourcePath, depPath)

		depModule := r.registry.Get(normalizedPath)
		if depModule == nil {
			continue
		}

		uid := r.getModuleUID(depModule)
		resolvedUIDs = append(resolvedUIDs, uid)
	}

	return resolvedUIDs, nil
}

func (r *Resolver) NormalizeDepPath(modulePath, depPath string) string {
	return r.normalizeDepPath(modulePath, depPath)
}

func (r *Resolver) normalizeDepPath(modulePath, depPath string) string {
	if filepath.IsAbs(depPath) {
		return NormalizedPath(depPath)
	}

	if !strings.HasPrefix(depPath, "library/") &&
		!strings.HasPrefix(depPath, "contrib/") &&
		!strings.HasPrefix(depPath, "yt/") &&
		!strings.HasPrefix(depPath, "arcadia/") {
		relDir := filepath.Dir(modulePath)
		absPath := filepath.Join(relDir, depPath)
		return NormalizedPath(absPath)
	}

	return NormalizedPath(depPath)
}

func (r *Resolver) getModuleUID(module *Module) string {
	uid := NewUID([]byte(module.SourcePath))
	return uid
}
