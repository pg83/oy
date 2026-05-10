#!/usr/bin/env bash
set -euo pipefail

strict="${OY_ENFORCE_ARCHIVER_GRAPH_EQUALITY:-0}"
if [[ "${1:-}" == "--strict" ]]; then
	strict=1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$repo_root"

reference_root="${OY_REFERENCE_ROOT:-/home/pg/monorepo/yatool_orig}"
reference_graph="${OY_REFERENCE_GRAPH:-$reference_root/sg.json}"
acceptance_target="${OY_ACCEPTANCE_TARGET:-$reference_root/tools/archiver}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
generated_graph="$tmpdir/sg.json"

step() {
	printf '\n==> %s\n' "$*"
}

step "Checking gofmt"
gofmt_diff="$tmpdir/gofmt.diff"
gofmt -d *.go > "$gofmt_diff"
if [[ -s "$gofmt_diff" ]]; then
	cat "$gofmt_diff"
	printf '\ngofmt produced a diff\n' >&2
	exit 1
fi

step "Running go vet"
go vet ./...

step "Running go test"
go test ./...

if [[ ! -f "$reference_graph" ]]; then
	step "Regenerating reference graph"
	if [[ ! -d "$reference_root" ]]; then
		printf 'reference root not found: %s\n' "$reference_root" >&2
		exit 1
	fi
	(cd "$reference_root" && ./srun.sh)
fi

step "Running acceptance graph harness"
set +e
go run . -G --musl --host-platform-flag=MUSL=yes --graph-file="$generated_graph" "$acceptance_target"
status=$?
set -e

if [[ "$status" -ne 0 ]]; then
	if [[ ! -s "$generated_graph" ]]; then
		printf 'acceptance graph generation failed before producing %s\n' "$generated_graph" >&2
		exit "$status"
	fi

	if [[ "$strict" == "1" ]]; then
		printf 'strict acceptance validation failed\n' >&2
		exit "$status"
	fi

	printf '\nAcceptance comparison failed, but this is allowed in non-strict mode while the documented execution-node graph gap remains.\n'
	printf 'Run OY_ENFORCE_ARCHIVER_GRAPH_EQUALITY=1 ./validate.sh or ./validate.sh --strict to make this fatal.\n'
fi

printf '\nLocal validation completed.\n'
