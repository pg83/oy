package main

import (
	"fmt"
	"strings"
)

type ParseFlagsResult struct {
	Ctx          *BuildContext
	PlatformFlag *PlatformFlags
}

func ParseFlags(args []string) (*ParseFlagsResult, error) {
	ctx := NewBuildContext()
	platformFlags := NewPlatformFlags()

	skipNext := false
	for i := 0; i < len(args); i++ {
		if skipNext {
			skipNext = false
			continue
		}

		arg := args[i]

		if arg == "--musl" {
			ctx.Musl = true
			continue
		}

		if strings.HasPrefix(arg, "--target-platform=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				ctx.TargetPlatform = parts[1]
			}
			continue
		}

		if strings.HasPrefix(arg, "--host-platform-flag=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				kv := strings.SplitN(parts[1], "=", 2)
				if len(kv) == 2 {
					platformFlags.SetHostFlag(kv[0], kv[1])
				}
			}
			continue
		}

		if strings.HasPrefix(arg, "--target-platform-flag=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				kv := strings.SplitN(parts[1], "=", 2)
				if len(kv) == 2 {
					platformFlags.SetTargetFlag(kv[0], kv[1])
				}
			}
			continue
		}

		if strings.HasPrefix(arg, "--language=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				ctx.Language = parts[1]
			}
			continue
		}
	}

	return &ParseFlagsResult{
		Ctx:          ctx,
		PlatformFlag: platformFlags,
	}, nil
}

type PlatformFlags struct {
	HostFlags   map[string]string
	TargetFlags map[string]string
}

func NewPlatformFlags() *PlatformFlags {
	return &PlatformFlags{
		HostFlags:   make(map[string]string),
		TargetFlags: make(map[string]string),
	}
}

func (p *PlatformFlags) SetHostFlag(key, value string) {
	p.HostFlags[key] = value
}

func (p *PlatformFlags) SetTargetFlag(key, value string) {
	p.TargetFlags[key] = value
}

func (p *PlatformFlags) GetHostFlag(key string) (string, bool) {
	val, ok := p.HostFlags[key]
	return val, ok
}

func (p *PlatformFlags) GetTargetFlag(key string) (string, bool) {
	val, ok := p.TargetFlags[key]
	return val, ok
}

func (p *PlatformFlags) ToMap() map[string]string {
	result := make(map[string]string)
	for k, v := range p.HostFlags {
		result[k] = v
	}
	for k, v := range p.TargetFlags {
		result[k] = v
	}
	return result
}

func (p *PlatformFlags) String() string {
	var hostStrs []string
	var targetStrs []string

	for k, v := range p.HostFlags {
		hostStrs = append(hostStrs, fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range p.TargetFlags {
		targetStrs = append(targetStrs, fmt.Sprintf("%s=%s", k, v))
	}

	return fmt.Sprintf("Host: [%s], Target: [%s]", strings.Join(hostStrs, ", "), strings.Join(targetStrs, ", "))
}
