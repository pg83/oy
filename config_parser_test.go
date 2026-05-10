package main

import (
	"testing"
)

func TestConfigParserMUSL(t *testing.T) {
	configPath := "/home/pg/monorepo/yatool_orig/build/ymake.core.conf"
	parser, err := ParseYmakeCoreConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	testVars := map[string]string{
		"MUSL":        "yes",
		"MUSL_LITE":   "no",
		"OS_LINUX":    "yes",
		"ARCH_X86_64": "yes",
	}

	injections := parser.ResolveConfigPEERDIRS(testVars)

	t.Logf("Test vars: %v", testVars)
	t.Logf("Resolved injections: %v", injections)

	foundMUSLFull := false
	for _, inj := range injections {
		if inj == "contrib/libs/musl/full" || inj == "contrib/libs/musl" {
			foundMUSLFull = true
			break
		}
	}

	if !foundMUSLFull {
		t.Errorf("Expected to find MUSL PEERDIR injection, got: %v", injections)
	}
}
