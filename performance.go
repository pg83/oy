package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

type PerformanceRun struct {
	Duration  time.Duration
	NodeCount int
}

type PerformanceReport struct {
	Runs      []PerformanceRun
	Median    time.Duration
	Commit    string
	GoVersion string
	OSKernel  string
	CPU       string
	Cores     int
	RAM       string
}

func MeasureGraphGeneration(targetPath string, ctx ParseContext, sourceRoot string, diag *TraversalLogger, runs int) PerformanceReport {
	if runs < 3 {
		runs = 3
	}
	if runs%2 == 0 {
		runs++
	}

	report := PerformanceReport{
		Runs:      make([]PerformanceRun, 0, runs),
		Commit:    bestEffortCommand("git", "rev-parse", "HEAD"),
		GoVersion: runtime.Version(),
		OSKernel:  bestEffortCommand("uname", "-a"),
		CPU:       bestEffortCPU(),
		Cores:     runtime.NumCPU(),
		RAM:       bestEffortRAM(),
	}

	for i := 0; i < runs; i++ {
		start := time.Now()
		graph := Throw2(BuildDependencyGraph(targetPath, ctx, sourceRoot, diag))
		duration := time.Since(start)

		report.Runs = append(report.Runs, PerformanceRun{Duration: duration, NodeCount: len(graph.Nodes)})
	}

	durations := make([]time.Duration, 0, len(report.Runs))
	for _, run := range report.Runs {
		durations = append(durations, run.Duration)
	}
	report.Median = medianDuration(durations)

	return report
}

func medianDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	values := make([]time.Duration, len(durations))
	copy(values, durations)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })

	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}

	return (values[middle-1] + values[middle]) / 2
}

func bestEffortCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "unknown"
	}

	value := strings.TrimSpace(stdout.String())
	if value == "" {
		return "unknown"
	}

	return value
}

func bestEffortCPU() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return "unknown"
}

func bestEffortRAM() string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "unknown"
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "MemTotal:"))
		}
	}

	return "unknown"
}
