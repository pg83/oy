# CLAUDE.md - ya/ymake Build System Reimplementation

This document provides a primer for new team members working on the ya/ymake build system reimplementation project.

## Project Overview

**Mission**: Reimplement the ya/ymake build system in Go with the goal of generating a bit-identical dependency graph in under 1 second.

**Repository**: This Go reimplementation
**Reference Implementation**: `/home/pg/monorepo/yatool_orig/` (Python/C++ original)

**Target**: `tools/archiver` - produce execution graph via `ya make -G tools/archiver`

**Key Requirements**:
- 100% graph structural match with reference implementation (modulo UID renumbering)
- Performance: graph generation in < 1 second on modern hardware
- Parallel parsing of includes and ya.make files
- Direct graph execution without JSON serialization intermediate
- Multimodule awareness (semantics depend on target and language flags)

## Directory Structure

This is a **flat Go project** - all `.go` files live in the repository root. No `internal/`, `cmd/`, or `pkg/` subdirectories.

```
/home/pg/monorepo/oy/
├── .go files (all in root)
├── throw.go           # Exception-style error handling
├── STYLE.md           # Code style conventions
├── GOALS.md           # Project goals (Russian, reference only)
├── run.sh             # Local build wrapper
└── claude             # CLI entry point
```

## Reference Implementation Locations

The reference ya/ymake system at `/home/pg/monorepo/yatool_orig/` contains:

| Path | Purpose |
|------|---------|
| `devtools/ymake/` | Graph generator (Python/C++) |
| `devtools/ya/` | Graph merger and orchestrator |
| `build/` | Build configuration (reference only, not to be interpreted) |
| `build/scripts/` | Scripts called from graph (OK to call directly) |
| `tools/archiver/ya.make` | Example target definition |
| `sg.json` | Reference graph output (3730 nodes) |

## Development Setup

**Requirements**: Go 1.21+ (any standard workstation)

**Building**:
```bash
go run *.go <target>
```

**Reference Graph Generation** (for validation):
```bash
cd /home/pg/monorepo/yatool_orig
./srun.sh  # generates sg.json for tools/archiver
```

## The ya.make DSL

ya.make is a text DSL that defines build modules with dependencies. The same file can have different interpretations depending on build flags (target platform, language, etc.).

### Module Types

```make
PROGRAM()    # Executable binary
LIBRARY()    # Static/shared library
END()        # Closes module definition
```

### Dependencies

```make
PEERDIR(
    library/cpp/archive
    library/cpp/digest/md5
)
```

PEERDIR declares dependency on another module. These form the graph edges.

### Source Files

```make
SRCS(
    main.cpp
    md5.cpp
    archive.cpp
)
```

Lists source files for the module.

### Includes

```make
RECURSE(
    ut
    bench
    medium_ut
)
```

Includes subdirectory ya.make files. Supports directory traversal for build modules.

### Platform Conditionals

```make
IF(MSVC)
    # Windows-specific configuration
ELSE()
    # Non-Windows configuration
ENDIF()

BUILD_ONLY_IF(OS_LINUX)
```

Conditional compilation based on platform or build flags. The same ya.make file may expose different module definitions depending on these conditions.

### Example: tools/archiver

```make
PROGRAM()

PEERDIR(
    library/cpp/archive
    library/cpp/digest/md5
    library/cpp/getopt/small
)

SRCS(
    main.cpp
)

SET(IDE_FOLDER "_Builders")

END()
```

## Graph Output Format

The build system outputs a JSON file with the following structure:

```json
{
  "conf": {
    "cache": true,
    "platform": "linux",
    "graph_size": 3730,
    "gsid": "USER:pg YA:...",
    ...
  },
  "graph": [
    {
      "uid": "bQglhGvE_E_M7mmKh1Nuwg",
      "self_uid": "tgeDq7dc6IWQFubGuYpjuA",
      "stats_uid": "c76f8ebdc20cd1d452491e62afe5aa78",
      "cmds": [...],
      "inputs": [...],
      "outputs": ["$(BUILD_ROOT)/tools/archiver/archiver"],
      "deps": [...],
      "kv": {...},
      "target_properties": {
        "module_dir": "tools/archiver",
        "module_lang": "cpp",
        "module_type": "bin"
      },
      ...
    },
    ...
  ],
  "inputs": [...],
  "result": [...]
}
```

**Node Keys**:
- `uid`: Node identifier (auto-generated, differs between runs)
- `self_uid`: Secondary identifier
- `cmds`: Array of command arrays (execution instructions)
- `inputs`: Input file paths (prefixed with `$(SOURCE_ROOT)`)
- `outputs`: Output file paths (prefixed with `$(BUILD_ROOT)`)
- `deps`: Dependency UIDs
- `target_properties`: Module metadata (dir, language, type)

Reference: `/home/pg/monorepo/yatool_orig/sg.json`

## Architecture Decisions

### Multimode Interpreter Design

ya.make semantics depend on input flags:
- **Target path**: Determines which module to build
- **Platform flags**: `--musl`, `--target-platform`, etc. change which branches evaluate
- **Language flags**: Affect proto and other language-specific modules

A single ya.make file may produce different graph nodes depending on these flags. The interpreter must evaluate conditionals at parse time based on the given context.

### Parallel Parsing Strategy

The build system must parse ya.make files concurrently:
- Include files (RECURSE) can be parsed in parallel
- Each ya.make file is independent until dependency resolution
- Use goroutines with shared graph structure (sync.Map or mutex-protected map)

Key insight: parsing is I/O-bound and embarrassingly parallel. Dependency resolution is the serial phase.

### In-Memory Graph Representation

**Goal**: Execute graph directly without JSON serialization.

**Approach**:
- Maintain graph nodes in Go structs in memory
- UIDs generated during graph construction
- Topological sort for execution order
- Direct invocation of commands via os/exec

This avoids the overhead of round-tripping through JSON between the graph generator and executor (as in the reference system).

## Code Style & Conventions

See `STYLE.md` for complete style guide. Key points:

### Error Handling

Use `throw.go` primitives instead of `if err != nil { return err }`:

```go
// BAD
f, err := os.Open(path)
if err != nil {
    return err
}

// GOOD
f := Throw2(os.Open(path))
```

Catches belong at boundaries (main, goroutine entries, filter loops).

### Formatting

- Blank lines before/after `if`, `for`, `switch`, `select` (except first/last statement)
- Blank line before `return` (except first statement)
- Logical grouping: consecutive one-liners stay together

### Project Layout

Flat `.go` files in repo root only.

### Dependencies

- **S3**: `github.com/aws/aws-sdk-go-v2/service/s3`
- **etcd**: `go.etcd.io/etcd/client/v3`
- **SSH**: Shell out to `ssh` binary via `exec.Command`
- **Config**: JSON only (no YAML)

## Acceptance Criteria

### Graph Equality (100%)

The generated graph must structurally match the reference graph, ignoring only UID renumbering.

**Validation approach**:
- Normalize UIDs (replace with sequential integers or hash-based IDs)
- Compare node counts, edges, command arrays
- Verify input/output paths match
- Ensure `target_properties` are identical

### Performance (< 1 second)

Target: `tools/archiver` graph generation in under 1 second on modern hardware.

**Measurement**:
```bash
time ya make -G tools/archiver > sg.json
```

**Optimization opportunities**:
- Parallel file parsing (goroutines)
- Avoid intermediate JSON in hot paths
- Cache parsed ya.make files in memory
- Use efficient graph structures (slice-based adjacency lists)

### Code Quality

No `go vet` or `golint` errors. Pass `throw.go` style checks.

**Linting**:
```bash
go vet ./...
golint .
```

### Test Coverage

Cover core DSL constructs:
- **PEERDIR**: Dependency declaration
- **SRCS**: Source file lists
- **RECURSE**: Include mechanism
- **IF/ELSE/ENDIF**: Platform conditionals

### CI Checklist

Before closing a PR:
1. Graph equality validation passes on `tools/archiver`
2. Performance benchmark shows < 1s (document hardware specs)
3. `go vet` and `golint` pass with zero errors
4. Unit tests for DSL parsers cover all constructs
5. Integration test generates bit-identical graph (modulo UIDs)

## Common Patterns

### Throwing Errors

```go
// File operations
content := Throw2(os.ReadFile(path))

// HTTP requests
resp := Throw2(http.Get(url))
defer resp.Body.Close()
body := Throw2(io.ReadAll(resp.Body))

// Disk operations
Throw(os.MkdirAll(dir, 0755))
```

### Parallel Parsing with Errgroup

```go
g, ctx := errgroup.WithContext(ctx)

for _, file := range files {
    file := file
    g.Go(func() error {
        content := Throw2(os.ReadFile(file))
        nodes := parseYaMake(content, ctx)
        graph.AddNodes(nodes)
        return nil
    })
}

Throw(g.Wait())
```

### Building Command Arrays

Graph nodes use command arrays:

```json
"cmds": [
  ["clang++", "-o", "output.o", "-c", "src.cpp", "-I", "include/"]
]
```

In Go:

```go
node.Cmds = [][]string{
    {"clang++", "-o", "output.o", "-c", "src.cpp", "-I", "include/"},
}
```

## Getting Started

1. Read `STYLE.md` and `throw.go` to understand the error handling pattern
2. Examine `/home/pg/monorepo/yatool_orig/tools/archiver/ya.make`
3. Generate the reference graph: `cd /home/pg/monorepo/yatool_orig && ./srun.sh`
4. Study `sg.json` structure (3730 nodes, ~100MB)
5. Implement a minimal ya.make parser for `PROGRAM()`, `LIBRARY()`, `PEERDIR()`, `SRCS()`, `END()`
6. Build graph for tools/archiver and validate against reference
7. Optimize for speed (parallel parsing, memory layout)

## Questions?

- Graph structure: See `sg.json` in reference implementation
- ya.make syntax: Examples throughout `/home/pg/monorepo/yatool_orig`
- Reference implementation: `devtools/ymake` (graph generation logic)
- Code style: Consult `STYLE.md` and existing files
