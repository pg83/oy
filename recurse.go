package main

import (
	"os"
	"path/filepath"
	"strings"
)

type RecurseProcessor struct {
	registry *ModuleRegistry
	ctx      *ParseContext
	visited  map[string]bool
}

func NewRecurseProcessor(registry *ModuleRegistry, ctx *ParseContext) *RecurseProcessor {
	return &RecurseProcessor{
		registry: registry,
		ctx:      ctx,
		visited:  make(map[string]bool),
	}
}

func (rp *RecurseProcessor) ProcessRecurse(module *Module, sourceRoot string) {
	if module == nil {
		return
	}

	for _, recurse := range module.Recursions {
		for _, relPath := range recurse.Paths {
			absPath := rp.resolveRecursePath(module.SourcePath, relPath, sourceRoot)
			rp.processDirectoryRecursive(absPath, sourceRoot)
		}
	}
}

func (rp *RecurseProcessor) ProcessFileImports(file *File, moduleDir string, sourceRoot string) {
	if file == nil {
		return
	}

	for _, recurse := range file.Imports {
		for _, relPath := range recurse.Paths {
			absPath := rp.resolveRecursePath(moduleDir, relPath, sourceRoot)
			rp.processDirectoryRecursive(absPath, sourceRoot)
		}
	}
}

func (rp *RecurseProcessor) resolveRecursePath(moduleDir, relPath, sourceRoot string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}

	absPath := filepath.Join(sourceRoot, moduleDir, relPath)
	return filepath.Clean(absPath)
}

func (rp *RecurseProcessor) processDirectoryRecursive(dirPath string, sourceRoot string) {
	normalizedPath := NormalizedPath(dirPath)

	if rp.visited[normalizedPath] {
		return
	}

	rp.visited[normalizedPath] = true

	fileInfo := Throw2(os.Stat(dirPath))
	if !fileInfo.IsDir() {
		return
	}

	yaMakePath := filepath.Join(dirPath, "ya.make")

	if _, err := os.Stat(yaMakePath); err == nil {
		rp.parseAndRegisterFile(yaMakePath, sourceRoot)
	}

	entries := Throw2(os.ReadDir(dirPath))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		subDirPath := filepath.Join(dirPath, entry.Name())

		subYaMake := filepath.Join(subDirPath, "ya.make")
		if _, err := os.Stat(subYaMake); err == nil {
			normalizedSubPath := NormalizedPath(subDirPath)
			if !rp.visited[normalizedSubPath] {
				rp.parseAndRegisterFile(subYaMake, sourceRoot)
			}
		}
	}
}

func (rp *RecurseProcessor) parseAndRegisterFile(yaMakePath string, sourceRoot string) {
	parser := NewParser(rp.ctx)

	file := Throw2(parser.ParseFile(yaMakePath))

	rp.ProcessFileImports(file, filepath.Dir(filepath.ToSlash(yaMakePath)), sourceRoot)

	moduleDir := filepath.Dir(filepath.ToSlash(yaMakePath))

	moduleDir = filepath.Join(sourceRoot, moduleDir)
	moduleDir = strings.TrimPrefix(moduleDir, sourceRoot)
	moduleDir = strings.TrimPrefix(moduleDir, "/")

	for _, module := range file.Modules {
		module.SourcePath = filepath.Join(moduleDir, module.SourcePath)

		rp.registry.Register(module.SourcePath, module)
	}
}
