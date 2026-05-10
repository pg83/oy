# CLAUDE.md - ya/ymake Build System Reimplementation

## Mission

`GOALS.md` is the authoritative project goal. This repository reimplements enough of ya/ymake in Go to produce a bit-identical execution graph for `/home/pg/monorepo/yatool_orig/tools/archiver`, ignoring only UID renumbering, with graph generation time no greater than 1 second.

Final completion requires 100% structural graph equality against the reference output and sub-second generation. Current implementation status is described below and does not yet satisfy the final graph-equality goal.

## Current Repository

This is a flat Go package: all `.go` source files live in the repository root. Do not introduce `cmd/`, `internal/`, or `pkg/` layouts unless the project direction changes.

`go.mod` currently declares `go 1.25.0`. The only declared module dependency is `golang.org/x/sync`.

`STYLE.md` is mandatory. `throw.go` provides exception-style helpers used instead of routine pass-through `if err != nil { return err }` error handling.

`run.sh` is not the local Go build command. It is an overseer/wirez wrapper:

```bash
exec subreaper /home/pg/monorepo/wirez/wirez ... -- /home/pg/monorepo/overseer/overseer "${@}"
```

## Reference Implementation

The reference ya/ymake checkout is `/home/pg/monorepo/yatool_orig`.

| Path | Purpose |
|------|---------|
| `devtools/ymake/` | Reference graph generator |
| `devtools/ya/` | Reference graph merger and orchestrator |
| `build/` | Reference build semantics input; use for understanding behavior, not as scripts to interpret wholesale |
| `build/scripts/` | Scripts called by graph nodes; direct calls are acceptable when required by generated commands |
| `tools/archiver/ya.make` | Reference target definition |
| `srun.sh` | Reference graph generation example for `tools/archiver` |
| `sg.json` | Reference graph output, currently 3730 nodes in the checked reference graph |

The Go implementation should encode the required semantics directly rather than depend on interpreting the full reference `build/` script layer.

## Commands

Local validation:

```bash
./validate.sh
./validate.sh --strict
```

`./validate.sh` is the supported local command for formatting, vet, tests, and the acceptance graph harness. `./validate.sh --strict` makes graph comparison mismatches fatal.

Individual checks run by local validation:

```bash
go test ./...
go vet ./...
gofmt -d *.go
```

Current CLI examples:

```bash
go run . /home/pg/monorepo/yatool_orig/tools/archiver
go run . -G --graph-file=test_sg.json /home/pg/monorepo/yatool_orig/tools/archiver
go run . --graph-file=test_sg.json /home/pg/monorepo/yatool_orig/tools/archiver
```

`go run . -G <target>` writes `sg.json` in the current working directory and prints progress lines to stdout before writing the graph file. Do not redirect stdout as if it were graph JSON.

Regenerate the reference graph with:

```bash
cd /home/pg/monorepo/yatool_orig && ./srun.sh
```

The current CLI resolves relative target paths against the workspace and also accepts absolute paths. This workspace does not contain `tools/archiver`, so use the absolute reference path for CLI smoke tests. The integration tests build `tools/archiver` by calling `BuildDependencyGraph` with source root `/home/pg/monorepo/yatool_orig`, so CLI parity with the reference source tree should be verified before documenting it as a final workflow.

## Current Status

Implemented scaffolding includes lexer/parser coverage, module registry behavior, conditionals, recurse handling, transitive dependency traversal, graph output, validation scaffolding, and integration tests for `tools/archiver`.

The current graph builder is module-level. It does not yet produce the reference 3730-node execution graph; current integration tests may skip the known node-count/equality gap. Final acceptance still requires a structural match against the full reference graph modulo UID renumbering.

## ya.make Semantics

`ya.make` is a text DSL that defines build modules, dependencies, sources, recursive includes, and conditional behavior. A single file can have different meanings depending on target path, platform flags, language flags, and variable context.

Common constructs:

```make
PROGRAM()
LIBRARY()
END()

PEERDIR(
    library/cpp/archive
    library/cpp/digest/md5
)

SRCS(
    main.cpp
)

RECURSE(
    ut
    bench
)

IF(MSVC)
    # Windows-specific configuration
ELSE()
    # Non-Windows configuration
ENDIF()

BUILD_ONLY_IF(OS_LINUX)
```

`PEERDIR` declares module dependencies. `SRCS` lists source files. `RECURSE` includes subdirectory `ya.make` files. `IF`/`ELSE`/`ENDIF` and `BUILD_ONLY_IF` are evaluated in the active build context.

## Architecture Requirements

The design must remain compatible with these `GOALS.md` requirements:

- Parallel parsing of includes and `ya.make` files.
- Direct graph execution from the in-memory representation without JSON serialization in the hot path.
- Multimodule and context-aware interpretation of `ya.make`, including target, platform, and language inputs.

Avoid over-prescribing implementation details in docs. Preserve room for changing graph data structures while keeping the acceptance contract stable.

## Style

Follow `STYLE.md` and existing repository conventions.

Use `throw.go` helpers for routine error propagation:

```go
content := Throw2(os.ReadFile(path))
Throw(os.MkdirAll(dir, 0755))
```

Catches belong at boundaries such as `main`, goroutine entries, or deliberate filter loops. Keep the flat root source layout.

## Validation

See `ACCEPTANCE.md` for the validation contract, current executable checks, known implementation gaps, graph equality expectations, and performance measurement rules.

## Binary Artifact Prevention

This repository uses a git pre-commit hook to prevent binary artifacts from being committed. To install the hook in your workspace:

```bash
./setup-precommit-hook.sh
```

The hook rejects:
- ELF executables, Mach-O binaries, PE executables
- Large files (>100KB) with non-whitelisted extensions
- Binary file formats (except known text types)

Allowed file types:
- `.go`, `.md`, `.json` (any size)
- `.txt`, `.sh`, `.mod`, `.sum`

To bypass the hook in emergencies, use `git commit --no-verify`.
