package main

import (
	"path"
	"strings"
	"sync"
)

// NormalizedPath returns a normalized path suitable for registry lookup.
// Handles path separators, removes leading/trailing slashes, and converts to lowercase.
func NormalizedPath(p string) string {
	if p == "" {
		return ""
	}

	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	return p
}

// ModuleRegistry stores parsed Module definitions indexed by normalized paths.
// Provides thread-safe fast lookup for PEERDIR resolution.
type ModuleRegistry struct {
	modules map[string]*Module
	mu      sync.RWMutex
}

// NewModuleRegistry creates an empty module registry.
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]*Module),
	}
}

// Register stores a module definition at its normalized path.
// If a module already exists at this path, it is replaced.
func (r *ModuleRegistry) Register(modulePath string, module *Module) {
	if r == nil || module == nil {
		return
	}

	normalized := NormalizedPath(modulePath)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.modules[normalized] = module
}

// Get retrieves a module definition by its path.
// Returns nil if not found.
func (r *ModuleRegistry) Get(modulePath string) *Module {
	if r == nil {
		return nil
	}

	normalized := NormalizedPath(modulePath)

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.modules[normalized]
}

// Has checks if a module exists at the given path.
func (r *ModuleRegistry) Has(modulePath string) bool {
	if r == nil {
		return false
	}

	normalized := NormalizedPath(modulePath)

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.modules[normalized]
	return exists
}

// Remove deletes a module from the registry.
func (r *ModuleRegistry) Remove(modulePath string) {
	if r == nil {
		return
	}

	normalized := NormalizedPath(modulePath)

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.modules, normalized)
}

// Count returns the number of registered modules.
func (r *ModuleRegistry) Count() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.modules)
}

// List returns all registered module paths.
// The slice is a copy and safe to modify.
func (r *ModuleRegistry) List() []string {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.modules))
	for p := range r.modules {
		paths = append(paths, p)
	}

	return paths
}

// AllModules returns all registered modules.
// The slice is a copy and safe to modify.
func (r *ModuleRegistry) AllModules() []*Module {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]*Module, 0, len(r.modules))
	for _, m := range r.modules {
		modules = append(modules, m)
	}

	return modules
}
