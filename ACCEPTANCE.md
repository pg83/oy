# ACCEPTANCE.md - ya/ymake Build System Reimplementation

This document defines the validation criteria for the ya/ymake build system reimplementation project. Each criterion must pass before the implementation is considered production-ready.

## Graph Equality Validation (CRITICAL)

The generated graph must structurally match the reference graph, ignoring only UID renumbering.

### Reference Data

- **Path**: `/home/pg/monorepo/yatool_orig/sg.json`
- **Node count**: 3730
- **Target**: `tools/archiver`
- **Reference generation command**: `cd /home/pg/monorepo/yatool_orig && ./srun.sh`

### Validation Procedure

1. Generate reference graph:
   ```bash
   cd /home/pg/monorepo/yatool_orig
   ./srun.sh
   ```

2. Generate test graph:
   ```bash
   go run *.go tools/archiver > test_sg.json
   ```

3. Compare using normalization:
   - Extract node fields: `uid`, `self_uid`, `stats_uid`, `cmds`, `inputs`, `outputs`, `deps`, `kv`, `target_properties`
   - Normalize UIDs: map each unique UID to sequential integer 0..N-1
   - Compare normalized graphs for structural equality

### Success Criteria

- **Matching node count**: 3730 nodes
- **Identical edge structure** after UID normalization
- **Matching command arrays** for each node
- **Identical input/output paths** (preserving `$(SOURCE_ROOT)` and `$(BUILD_ROOT)` prefixes)
- **Matching `target_properties`**:
  - `module_dir`: module directory path
  - `module_lang`: programming language (e.g., "cpp")
  - `module_type`: module type (e.g., "bin", "lib")
- **Zero extraneous or missing nodes**

### UID Normalization Algorithm

Since UIDs are auto-generated and differ between runs, normalize before comparison:

1. Extract all unique UIDs from both graphs
2. Create a mapping: `original_uid -> sequential_integer` (0, 1, 2, ...)
3. Replace all UID references (`uid`, `self_uid`, `stats_uid`, `deps[]`) using the mapping
4. Compare normalized JSON structures for equality

## Performance Validation (CRITICAL)

Target: `tools/archiver` graph generation in under 1 second on modern hardware.

### Measurement Command

```bash
time go run *.go tools/archiver > /dev/null
```

### Success Criteria

- **Wall clock time**: < 1.0 second on reference hardware
- **Warm cache**: Run 3 times, take median value
- **Hardware baseline**: Modern workstation (document specs in validation notes)

### Benchmark Script

```bash
#!/bin/bash
# perf_validation.sh

echo "Performance validation - tools/archiver"
echo "Hardware: $(uname -m) CPUs: $(nproc) RAM: $(free -h | grep Mem | awk '{print $2}')"
echo ""

for i in {1..3}; do
    echo "Run $i:"
    /usr/bin/time -f "%E real" go run *.go tools/archiver > /dev/null 2>&1
done

echo ""
echo "Median should be < 1.0 second"
```

### Hardware Baseline

Document your hardware specs when running validation:

```
CPU: <model> @ <frequency>
Cores: <number>
RAM: <size>
OS: <distribution>
Go version: <output of go version>
```

Example reference baseline:
```
CPU: Intel Core i7-12700K @ 3.60GHz
Cores: 12 (8P + 4E)
RAM: 32GB DDR5-4800
OS: Ubuntu 22.04 LTS
Go version: go1.21.5 linux/amd64
```

### Optimization Opportunities

If performance targets are not met, consider:

- Parallel file parsing (goroutines)
- Avoid intermediate JSON in hot paths
- Cache parsed ya.make files in memory
- Use efficient graph structures (slice-based adjacency lists)

## Code Quality Validation (MEDIUM)

### Check Commands

```bash
go vet ./...
golint .
```

### Success Criteria

- **Zero warnings** from `go vet`
- **Zero warnings** from `golint`
- **No `if err != nil { return err }` patterns** (use `throw.go` primitives)
- **Proper blank lines** around control blocks (see STYLE.md)
- **Blank lines before `return` statements** (except first statement)

### Error Handling Validation

All error handling must use `throw.go` primitives:

```go
// BAD
f, err := os.Open(path)
if err != nil {
    return err
}

// GOOD
f := Throw2(os.Open(path))
```

Allowed `if err != nil` patterns:
- Filter: `if err != nil { continue }` (skip bad items in loops)
- Discriminate: `errors.As`/`errors.Is` checks for expected failures (e.g., exit codes, HTTP 404)

## Test Coverage Validation (MEDIUM)

### Core DSL Constructs to Test

- `PROGRAM()` - Executable binary module type
- `LIBRARY()` - Static/shared library module type
- `END()` - Module definition closure
- `PEERDIR()` - Dependency declaration
- `SRCS()` - Source file lists
- `RECURSE()` - Include mechanism for subdirectory ya.make files
- `IF/ELSE/ENDIF` - Platform conditionals

### Test Organization

- **Unit tests**: Individual parsers for each DSL construct
- **Integration tests**: Complete ya.make files with multiple constructs
- **Golden file tests**: Graph output compared against reference graphs
- **Performance tests**: Validate sub-second graph generation

### Success Criteria

- **All DSL constructs covered** by at least one test
- **Golden graph passes equality validation** on 3730-node reference graph
- **Test execution time**: < 5 seconds total
- **Zero flaky tests**

### Test Execution

```bash
go test ./...
```

## CI/CD Checklist

Before merging any changes to trunk:

### Pre-merge Validation

- [ ] Graph equality validation passes on `tools/archiver`
  - Reference graph: `/home/pg/monorepo/yatool_orig/sg.json`
  - Test graph matches modulo UID renumbering
  - Node count: 3730

- [ ] Performance benchmark shows < 1s
  - Document hardware specs in validation notes
  - Median of 3 warm runs < 1.0 second

- [ ] `go vet` and `golint` pass with zero errors
  - `go vet ./...` returns no warnings
  - `golint .` returns no warnings

- [ ] Unit tests for DSL parsers cover all constructs
  - PROGRAM, LIBRARY, PEERDIR, SRCS, RECURSE, IF/ELSE/ENDIF
  - Test execution time < 5 seconds

- [ ] Integration test generates bit-identical graph (modulo UIDs)
  - Golden file test passes
  - No missing or extraneous nodes

- [ ] Code follows STYLE.md conventions
  - Error handling uses throw.go primitives
  - Proper blank lines around control blocks
  - Blank lines before return statements

- [ ] All error handling uses throw.go primitives
  - No `if err != nil { return err }` pass-through patterns
  - Exceptions only at boundaries (main, goroutine entries, filter loops)

### Post-merge Verification

- [ ] Trunk builds successfully
- [ ] Full test suite passes on CI infrastructure
- [ ] No regressions in performance benchmarks

## Appendices

### Appendix A: Reference Graph Schema

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
      "cmds": [
        ["clang++", "-o", "output.o", "-c", "src.cpp", "-I", "include/"]
      ],
      "inputs": [
        "$(SOURCE_ROOT)/path/to/source.cpp"
      ],
      "outputs": [
        "$(BUILD_ROOT)/path/to/output.o"
      ],
      "deps": [
        "dep_uid_1",
        "dep_uid_2"
      ],
      "kv": {
        "key": "value",
        ...
      },
      "target_properties": {
        "module_dir": "tools/archiver",
        "module_lang": "cpp",
        "module_type": "bin"
      },
      ...
    }
  ],
  "inputs": [...],
  "result": [...]
}
```

### Appendix B: UID Normalization Implementation

```go
func normalizeGraph(graph []*GraphNode) {
    // Collect all unique UIDs
    uidMap := make(map[string]int)
    nextID := 0

    // First pass: assign sequential IDs to all UIDs
    for _, node := range graph {
        if _, exists := uidMap[node.UID]; !exists {
            uidMap[node.UID] = nextID
            nextID++
        }
        if _, exists := uidMap[node.SelfUID]; !exists {
            uidMap[node.SelfUID] = nextID
            nextID++
        }
        if _, exists := uidMap[node.StatsUID]; !exists {
            uidMap[node.StatsUID] = nextID
            nextID++
        }
        for _, dep := range node.Deps {
            if _, exists := uidMap[dep]; !exists {
                uidMap[dep] = nextID
                nextID++
            }
        }
    }

    // Second pass: replace UIDs with sequential integers
    for _, node := range graph {
        node.UID = fmt.Sprintf("uid_%d", uidMap[node.UID])
        node.SelfUID = fmt.Sprintf("self_%d", uidMap[node.SelfUID])
        node.StatsUID = fmt.Sprintf("stats_%d", uidMap[node.StatsUID])
        for i, dep := range node.Deps {
            node.Deps[i] = fmt.Sprintf("uid_%d", uidMap[dep])
        }
    }
}
```

### Appendix C: Example Validation Script

```bash
#!/bin/bash
# validate_graph.sh

set -e

REF_DIR="/home/pg/monorepo/yatool_orig"
REF_GRAPH="$REF_DIR/sg.json"
TEST_GRAPH="test_sg.json"

echo "=== Graph Equality Validation ==="

# Generate test graph
echo "Generating test graph..."
go run *.go tools/archiver > "$TEST_GRAPH"

# Compare node counts
REF_NODES=$(jq '.conf.graph_size' "$REF_GRAPH")
TEST_NODES=$(jq '.conf.graph_size' "$TEST_GRAPH")

echo "Reference nodes: $REF_NODES"
echo "Test nodes: $TEST_NODES"

if [ "$REF_NODES" -ne "$TEST_NODES" ]; then
    echo "FAIL: Node count mismatch"
    exit 1
fi

# Structural comparison (simplified - actual implementation needs UID normalization)
echo "Node count matches. Full structural comparison requires UID normalization."

echo "=== Performance Validation ==="
for i in {1..3}; do
    echo "Run $i:"
    /usr/bin/time -f "%E real" go run *.go tools/archiver > /dev/null 2>&1
done

echo "=== Code Quality Validation ==="
echo "Running go vet..."
go vet ./...

echo "Running golint..."
golint .

echo "=== All validations passed ==="
```

### Appendix D: Hardware Baseline Template

When recording performance results, use this template:

```
Hardware baseline:
- CPU: <manufacturer> <model> @ <clock speed>
- Cores: <physical> (P-cores: <count>, E-cores: <count>)
- RAM: <size> <type>-<speed>
- Storage: <type> (SSD/NVMe)
- OS: <distribution> <kernel version>
- Go version: <version> <platform>

Performance results (tools/archiver):
- Run 1: <time>
- Run 2: <time>
- Run 3: <time>
- Median: <time>
- Pass/Fail: <result>
```

## References

- Project overview: `CLAUDE.md`
- Code style: `STYLE.md`
- Reference implementation: `/home/pg/monorepo/yatool_orig/`
- Reference graph: `/home/pg/monorepo/yatool_orig/sg.json`
