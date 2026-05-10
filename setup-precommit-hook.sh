#!/usr/bin/env bash
set -euo pipefail

if [[ -f .git/hooks/pre-commit ]]; then
    echo "Pre-commit hook already exists at .git/hooks/pre-commit"
    echo "Skipping installation to avoid overwriting existing hook"
    exit 0
fi

mkdir -p .git/hooks

cat << 'EOF' > .git/hooks/pre-commit
#!/usr/bin/env bash
set -euo pipefail

staged_files=$(git diff --cached --name-only --diff-filter=ACMR)

if [[ -z "$staged_files" ]]; then
    exit 0
fi

violations=()

for file in $staged_files; do
    if [[ ! -f "$file" ]]; then
        continue
    fi

    size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
    if [[ "$size" -gt 102400 ]]; then
        ext="${file##*.}"
        if [[ "$ext" != "json" && "$ext" != "md" && "$ext" != "txt" && "$ext" != "go" ]]; then
            violations+=("$file: Large file ($size bytes, extension .$ext not allowed for large commits)")
            continue
        fi
    fi

    file_type=$(file -b "$file" 2>/dev/null || echo "unknown")

    if echo "$file_type" | grep -qE "ELF.*executable|Mach-O.*executable|PE32 executable|MS-DOS executable"; then
        violations+=("$file: Binary executable ($file_type) not allowed in repository")
        continue
    fi

    if echo "$file_type" | grep -qEv "ASCII text|UTF-8|Unicode text|text"; then
        if ! [[ "$file_type" =~ (data|compressed) ]]; then
            violations+=("$file: Detected as binary ($file_type), only text files allowed")
            continue
        fi
    fi
done

if [[ ${#violations[@]} -gt 0 ]]; then
    echo "ERROR: Pre-commit hook blocked commit due to binary artifacts:" >&2
    for violation in "${violations[@]}"; do
        echo "  - $violation" >&2
    done
    echo "" >&2
    echo "To commit these files, add them to .gitignore or use --no-verify to bypass (not recommended)" >&2
    exit 1
fi

exit 0
EOF

chmod +x .git/hooks/pre-commit

echo "Pre-commit hook installed successfully at .git/hooks/pre-commit"
echo "The hook will reject binary artifacts (ELF executables, Mach-O binaries, PE executables)"
echo "and large non-text files while allowing legitimate text files (.go, .md, .json)"
