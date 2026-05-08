package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// NormalizeGraph applies the rules in acceptance.md §4 to a raw sg.json-shaped
// document and returns its canonical JSON encoding. It Throws on malformed
// input, cycles, or dangling deps.
func NormalizeGraph(raw []byte) []byte {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var top map[string]any
	Throw(dec.Decode(&top))

	if conf, ok := top["conf"].(map[string]any); ok {
		stripConfVolatile(conf)
	}

	graph := castNodeArray(top["graph"])
	result := castStringArray(top["result"])

	for _, n := range graph {
		dropEmptyOptional(n)
	}

	mapping := computeShapes(graph)
	applyMapping(graph, result, mapping)
	sortGraph(graph)
	sort.Strings(result)

	top["graph"] = nodesToAny(graph)
	top["result"] = stringsToAny(result)

	return canonicalJSON(top)
}

func stripConfVolatile(conf map[string]any) {
	delete(conf, "gsid")

	if d, ok := conf["description"].(map[string]any); ok {
		delete(d, "host")
		delete(d, "platform")
		delete(d, "user")
	}

	if rs, ok := conf["resources"].([]any); ok {
		for _, e := range rs {
			r, ok := e.(map[string]any)

			if !ok {
				continue
			}

			if _, present := r["resource"]; present {
				r["resource"] = "<vcs-resource>"
			}
		}
	}
}

func dropEmptyOptional(node map[string]any) {
	for _, k := range []string{
		"host_platform",
		"foreign_deps",
		"env",
		"tags",
		"requirements",
		"target_properties",
		"kv",
	} {
		v, present := node[k]

		if !present {
			continue
		}

		if isEmpty(v) {
			delete(node, k)
		}
	}
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}

	return false
}

func computeShapes(graph []map[string]any) map[string]string {
	byUID := make(map[string]map[string]any, len(graph))

	for _, n := range graph {
		u, ok := n["uid"].(string)

		if !ok {
			ThrowFmt("node missing uid")
		}

		byUID[u] = n
	}

	shapes := make(map[string]string, len(graph))
	state := make(map[string]int8, len(graph))

	var visit func(u string) string

	visit = func(u string) string {
		switch state[u] {
		case 1:
			ThrowFmt("cycle detected at uid %s", u)
		case 2:
			return shapes[u]
		}

		state[u] = 1

		n, ok := byUID[u]

		if !ok {
			ThrowFmt("dangling dep uid %s", u)
		}

		deps := castStringArray(n["deps"])
		depShapes := make([]string, 0, len(deps))

		for _, d := range deps {
			depShapes = append(depShapes, visit(d))
		}

		sort.Strings(depShapes)

		h := sha256.New()
		h.Write(contentBytesWithoutUIDs(n))

		for _, ds := range depShapes {
			h.Write([]byte{0})
			h.Write([]byte(ds))
		}

		sig := "U:" + hex.EncodeToString(h.Sum(nil)[:8])
		shapes[u] = sig
		state[u] = 2

		return sig
	}

	for _, n := range graph {
		visit(n["uid"].(string))
	}

	return shapes
}

func contentBytesWithoutUIDs(node map[string]any) []byte {
	clone := make(map[string]any, len(node))

	for k, v := range node {
		switch k {
		case "uid", "self_uid", "stats_uid", "deps", "foreign_deps":
			continue
		}

		clone[k] = v
	}

	return canonicalJSON(clone)
}

func applyMapping(graph []map[string]any, result []string, mapping map[string]string) {
	for _, n := range graph {
		u := n["uid"].(string)
		canonical, ok := mapping[u]

		if !ok {
			ThrowFmt("uid %s missing from mapping", u)
		}

		n["uid"] = canonical

		if _, present := n["self_uid"]; present {
			n["self_uid"] = canonical
		}

		if _, present := n["stats_uid"]; present {
			n["stats_uid"] = canonical
		}

		if deps, present := n["deps"]; present {
			n["deps"] = remapStringArray(deps, mapping, false)
		}

		if fd, present := n["foreign_deps"]; present {
			n["foreign_deps"] = remapForeignDeps(fd, mapping)
		}
	}

	for i, r := range result {
		canonical, ok := mapping[r]

		if !ok {
			ThrowFmt("result uid %s missing from mapping", r)
		}

		result[i] = canonical
	}
}

// remapStringArray rewrites every uid in src through mapping and returns a
// sorted []any. When allowExternal is true, uids not in the mapping are
// kept unchanged — used for foreign_deps, which legitimately reference
// nodes in other graphs (host-platform tools etc.). For local edges
// (deps, result) a missing uid is a hard error.
func remapStringArray(v any, mapping map[string]string, allowExternal bool) []any {
	src := castStringArray(v)
	out := make([]any, 0, len(src))

	for _, s := range src {
		canonical, ok := mapping[s]

		if !ok {
			if !allowExternal {
				ThrowFmt("dep uid %s missing from mapping", s)
			}

			canonical = s
		}

		out = append(out, canonical)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].(string) < out[j].(string)
	})

	return out
}

func remapForeignDeps(v any, mapping map[string]string) map[string]any {
	src, ok := v.(map[string]any)

	if !ok {
		ThrowFmt("foreign_deps is not an object")
	}

	out := make(map[string]any, len(src))

	for bucket, arr := range src {
		out[bucket] = remapStringArray(arr, mapping, true)
	}

	return out
}

func sortGraph(graph []map[string]any) {
	sort.SliceStable(graph, func(i, j int) bool {
		return graph[i]["uid"].(string) < graph[j]["uid"].(string)
	})
}

func castNodeArray(v any) []map[string]any {
	arr, ok := v.([]any)

	if !ok {
		ThrowFmt("graph is not an array")
	}

	out := make([]map[string]any, 0, len(arr))

	for i, e := range arr {
		n, ok := e.(map[string]any)

		if !ok {
			ThrowFmt("graph[%d] is not an object", i)
		}

		out = append(out, n)
	}

	return out
}

func castStringArray(v any) []string {
	if v == nil {
		return nil
	}

	arr, ok := v.([]any)

	if !ok {
		ThrowFmt("expected string array, got %T", v)
	}

	out := make([]string, 0, len(arr))

	for i, e := range arr {
		s, ok := e.(string)

		if !ok {
			ThrowFmt("array element [%d] is not a string", i)
		}

		out = append(out, s)
	}

	return out
}

func nodesToAny(graph []map[string]any) []any {
	out := make([]any, len(graph))

	for i, n := range graph {
		out[i] = n
	}

	return out
}

func stringsToAny(ss []string) []any {
	out := make([]any, len(ss))

	for i, s := range ss {
		out[i] = s
	}

	return out
}

// canonicalJSON marshals v with sorted object keys and a two-space indent,
// preserving json.Number values to avoid float reformatting. The result is
// terminated with a single trailing newline.
func canonicalJSON(v any) []byte {
	var buf bytes.Buffer
	writeCanonical(&buf, v, 0)
	buf.WriteByte('\n')

	return buf.Bytes()
}

func writeCanonical(buf *bytes.Buffer, v any, indent int) {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		writeJSONString(buf, x)
	case json.Number:
		buf.WriteString(string(x))
	case float64:
		buf.WriteString(strconv.FormatFloat(x, 'g', -1, 64))
	case int:
		buf.WriteString(strconv.FormatInt(int64(x), 10))
	case int64:
		buf.WriteString(strconv.FormatInt(x, 10))
	case map[string]any:
		writeCanonicalObject(buf, x, indent)
	case []any:
		writeCanonicalArray(buf, x, indent)
	default:
		ThrowFmt("canonicalJSON: unsupported type %T", v)
	}
}

func writeCanonicalObject(buf *bytes.Buffer, m map[string]any, indent int) {
	if len(m) == 0 {
		buf.WriteString("{}")

		return
	}

	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	buf.WriteByte('{')
	buf.WriteByte('\n')
	pad := strings.Repeat("  ", indent+1)

	for i, k := range keys {
		buf.WriteString(pad)
		writeJSONString(buf, k)
		buf.WriteString(": ")
		writeCanonical(buf, m[k], indent+1)

		if i < len(keys)-1 {
			buf.WriteByte(',')
		}

		buf.WriteByte('\n')
	}

	buf.WriteString(strings.Repeat("  ", indent))
	buf.WriteByte('}')
}

func writeCanonicalArray(buf *bytes.Buffer, a []any, indent int) {
	if len(a) == 0 {
		buf.WriteString("[]")

		return
	}

	buf.WriteByte('[')
	buf.WriteByte('\n')
	pad := strings.Repeat("  ", indent+1)

	for i, e := range a {
		buf.WriteString(pad)
		writeCanonical(buf, e, indent+1)

		if i < len(a)-1 {
			buf.WriteByte(',')
		}

		buf.WriteByte('\n')
	}

	buf.WriteString(strings.Repeat("  ", indent))
	buf.WriteByte(']')
}

func writeJSONString(buf *bytes.Buffer, s string) {
	b := Throw2(json.Marshal(s))
	buf.Write(b)
}
