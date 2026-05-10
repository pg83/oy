package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetNodeDistribution(t *testing.T) {
	ctx := ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: ReferenceTarget,
		BuildFlags: make(map[string]string),
	}

	graph := Throw2(BuildDependencyGraph(ReferenceTarget, ctx, SourceRoot, nil))

	counts := make(map[string]int)
	for _, node := range graph.Nodes {
		cmdType := ""
		if len(node.Cmds) > 0 {
			cmd := node.Cmds[0]
			if len(cmd.CmdArgs) > 0 {
				cmd0 := strings.TrimPrefix(cmd.CmdArgs[0], "$(CLANG-2403293607)/bin/")
				cmd0 = strings.TrimPrefix(cmd0, "$(BUILD_ROOT)/contrib/tools/yasm/")
				cmd0 = strings.TrimPrefix(cmd0, "$(YMAKE_PYTHON3-1002064631)/bin/")
				switch cmd0 {
				case "clang++", "clang", "g++", "gcc", "c++", "cc":
					cmdType = "CC"
				case "llvm-as":
					cmdType = "AS"
				case "ar":
					cmdType = "AR"
				case "ld", "ld.lld":
					cmdType = "LD"
				case "ranlib":
					cmdType = "R6"
				case "python3", "python", "yasm", "ragel6":
					cmdType = "JSAS"
				default:
					cmdType = "OTHER"
				}
			}
		} else if node.KV["graph_node_type"] == "JSENV" {
			cmdType = "JSAS"
		} else {
			cmdType = "OTHER"
		}
		counts[cmdType]++
	}

	fmt.Println("\n=== Current Implementation Node Distribution ===")
	fmt.Printf("Total nodes: %d\n", len(graph.Nodes))
	fmt.Printf("CC nodes: %d\n", counts["CC"])
	fmt.Printf("JS/AS nodes (python/yasm/ragel): %d\n", counts["JSAS"])
	fmt.Printf("Other nodes: %d\n", counts["OTHER"])

	fmt.Println("\n=== Reference Distribution (from sg.json) ===")
	fmt.Println("Total nodes: 3730")
	fmt.Println("CC nodes (clang/clang++): 3203 + 426 = 3629")
	fmt.Println("JS/AS nodes (python/yasm/ragel): 75 + 25 + 1 = 101")
	fmt.Println("Missing JS/AS implementation: 101 nodes")
}
