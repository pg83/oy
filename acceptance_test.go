package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func referencePath(t *testing.T) string {
	p := os.Getenv("YATOOL_ORIG")

	if p == "" {
		p = "/home/pg/monorepo/yatool_orig"
	}

	sg := filepath.Join(p, "sg.json")

	if _, err := os.Stat(sg); err != nil {
		t.Skipf("reference fixture not available at %s: %v", sg, err)
	}

	return p
}

func budgetMS() int64 {
	v := os.Getenv("ACCEPTANCE_TIME_BUDGET_MS")

	if v == "" {
		return 1000
	}

	return Throw2(strconv.ParseInt(v, 10, 64))
}

func TestNormalizeIdempotent(t *testing.T) {
	Try(func() {
		ya := referencePath(t)
		raw := Throw2(os.ReadFile(filepath.Join(ya, "sg.json")))

		a := NormalizeGraph(raw)
		b := NormalizeGraph(a)

		if !bytes.Equal(a, b) {
			t.Fatalf("normalize is not idempotent (a=%d bytes, b=%d bytes)",
				len(a), len(b))
		}
	}).Catch(func(e *Exception) {
		t.Fatalf("%v", e)
	})
}

func TestNormalizeReferenceSelfDiff(t *testing.T) {
	Try(func() {
		ya := referencePath(t)
		raw := Throw2(os.ReadFile(filepath.Join(ya, "sg.json")))

		n := NormalizeGraph(raw)

		if !bytes.Equal(n, NormalizeGraph(raw)) {
			t.Fatal("reference does not normalize stably")
		}
	}).Catch(func(e *Exception) {
		t.Fatalf("%v", e)
	})
}

func TestAcceptanceArchiver(t *testing.T) {
	Try(func() {
		ya := referencePath(t)

		if BuildArchiverGraph == nil {
			t.Skip("BuildArchiverGraph not implemented yet")
		}

		start := time.Now()
		ours := Throw2(BuildArchiverGraph(ya))
		elapsed := time.Since(start)
		t.Logf("BuildArchiverGraph wall time: %v", elapsed)

		ref := Throw2(os.ReadFile(filepath.Join(ya, "sg.json")))

		a := NormalizeGraph(ours)
		b := NormalizeGraph(ref)

		if !bytes.Equal(a, b) {
			dumpDiff(t, a, b)
			t.Fatal("graph differs from reference (modulo uid renumbering)")
		}

		budget := time.Duration(budgetMS()) * time.Millisecond

		if elapsed > budget {
			t.Fatalf("wall time %v exceeds budget %v", elapsed, budget)
		}
	}).Catch(func(e *Exception) {
		t.Fatalf("%v", e)
	})
}

const diffLineCap = 400

// dumpDiff prints a truncated line-based diff between two canonical-JSON
// byte slices. It walks the two side by side and emits +/- markers wherever
// they diverge, capped at diffLineCap lines, and finishes with a summary
// line counting only-in-A, only-in-B, and shared lines.
func dumpDiff(t *testing.T, a, b []byte) {
	la := strings.Split(string(a), "\n")
	lb := strings.Split(string(b), "\n")

	emitted := 0
	onlyA, onlyB, shared := 0, 0, 0
	i, j := 0, 0

	for i < len(la) && j < len(lb) {
		if la[i] == lb[j] {
			shared++
			i++
			j++

			continue
		}

		if emitted < diffLineCap {
			t.Logf("- %s", la[i])
			emitted++
		}

		if emitted < diffLineCap {
			t.Logf("+ %s", lb[j])
			emitted++
		}

		onlyA++
		onlyB++
		i++
		j++
	}

	for ; i < len(la); i++ {
		if emitted < diffLineCap {
			t.Logf("- %s", la[i])
			emitted++
		}

		onlyA++
	}

	for ; j < len(lb); j++ {
		if emitted < diffLineCap {
			t.Logf("+ %s", lb[j])
			emitted++
		}

		onlyB++
	}

	t.Logf("diff summary: only-in-ours=%d only-in-ref=%d shared=%d truncated_at=%d",
		onlyA, onlyB, shared, diffLineCap)
}
