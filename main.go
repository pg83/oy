package main

import (
	"fmt"
	"os"
)

func main() {

	if exc := Try(func() {

		if len(os.Args) > 1 && os.Args[1] == "lex" {
			lexToolMain()
			return
		}

		result := Throw2(ParseFlags(os.Args))

		vars := NewMemoryVariableSet()

		for k, v := range result.PlatformFlag.ToMap() {
			vars.SetValue(k, v)
		}

		fullVars := NewBuildContextVariableSet(result.Ctx, vars)

		fmt.Printf("ymake - Ya Make build system reimplementation\n")
		fmt.Printf("Build context: Musl=%v, TargetPlatform=%s, Language=%s\n", result.Ctx.Musl, result.Ctx.TargetPlatform, result.Ctx.Language)
		fmt.Printf("Platform flags: %s\n", result.PlatformFlag.String())

		if len(os.Args) > 1 {
			fmt.Printf("Args: %v\n", os.Args)
			if fullVars.IsTrue("MUSL") {
				fmt.Printf("MUSL flag is enabled\n")
			}
			if result.Ctx.TargetPlatform != "" {
				targetPlatformKey := "TARGET_PLATFORM_DEFAULT_LINUX_AARCH64"
				fmt.Printf("Checking %s: %v\n", targetPlatformKey, fullVars.IsTrue(targetPlatformKey))
			}
		}

	}); exc != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", exc.AsError())
		os.Exit(1)
	}

}
