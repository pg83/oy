package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	kwWhen      = "when"
	kwOtherwise = "otherwise"
	kwPeerdir   = "PEERDIR"
)

type ConfigPEERDIRInjection struct {
	Variable    string
	ExpectedVal string
	Peerdirs    []string
}

type ConfigParser struct {
	injections map[string][]*ConfigPEERDIRInjection
	variables  map[string]string
}

type Condition struct {
	Variable    string
	ExpectedVal string
	Negated     bool
}

func NewConfigParser() *ConfigParser {
	return &ConfigParser{
		injections: make(map[string][]*ConfigPEERDIRInjection),
		variables:  make(map[string]string),
	}
}

func ParseYmakeCoreConfig(configPath string) (*ConfigParser, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	content := Throw2(os.ReadFile(configPath))
	parser := NewConfigParser()

	lines := strings.Split(string(content), "\n")
	parser.parseBlocks(lines)

	return parser, nil
}

func (cp *ConfigParser) parseBlocks(lines []string) {
	whenPattern := regexp.MustCompile(`^when\s*\(\$(\w+)\s*==\s*\"([^"]+)\"\)\s*\{`)
	notWhenPattern := regexp.MustCompile(`^when\s*\(\$(\w+)\s*!=\s*\"([^"]+)\"\)\s*\{`)
	peerdirPattern := regexp.MustCompile(`^\s*PEERDIR\s*\+=(.+)`)

	var blockStack []Condition
	var blockContent []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if whenMatch := whenPattern.FindStringSubmatch(line); len(whenMatch) > 0 {
			blockStack = append(blockStack, Condition{
				Variable:    whenMatch[1],
				ExpectedVal: whenMatch[2],
				Negated:     false,
			})
			blockContent = []string{}
			continue
		}

		if notWhenMatch := notWhenPattern.FindStringSubmatch(line); len(notWhenMatch) > 0 {
			blockStack = append(blockStack, Condition{
				Variable:    notWhenMatch[1],
				ExpectedVal: notWhenMatch[2],
				Negated:     true,
			})
			blockContent = []string{}
			continue
		}

		if strings.HasPrefix(line, "otherwise {") || strings.HasPrefix(line, kwOtherwise) {
			if len(blockStack) > 0 {
				blockStack[len(blockStack)-1].Negated = !blockStack[len(blockStack)-1].Negated
			}
			continue
		}

		if strings.Contains(line, "}") && len(blockStack) > 0 {
			if len(blockContent) > 0 {
				cp.processNestedBlocks(blockStack, blockContent)
			}
			blockStack = blockStack[:len(blockStack)-1]
			blockContent = []string{}
			continue
		}

		if len(blockStack) > 0 {
			if match := peerdirPattern.FindStringSubmatch(line); len(match) > 0 {
				peerdirs := strings.Split(strings.TrimSpace(match[1]), " ")
				peerdirs = filterEmptyStrings(peerdirs)
				blockContent = append(blockContent, strings.Join(peerdirs, " "))
			}
			continue
		}

		if match := peerdirPattern.FindStringSubmatch(line); len(match) > 0 {
			peerdirs := strings.Split(strings.TrimSpace(match[1]), " ")
			peerdirs = filterEmptyStrings(peerdirs)
			for _, pd := range peerdirs {
				cp.addGlobalInjection(pd)
			}
		}
	}
}

func (cp *ConfigParser) processNestedBlocks(conditions []Condition, content []string) {
	for _, line := range content {
		peerdirs := strings.Split(line, " ")
		peerdirs = filterEmptyStrings(peerdirs)
		if len(peerdirs) == 0 {
			continue
		}

		for _, cond := range conditions {
			if cond.Negated {
				continue
			}
			injection := &ConfigPEERDIRInjection{
				Variable:    cond.Variable,
				ExpectedVal: cond.ExpectedVal,
				Peerdirs:    peerdirs,
			}
			cp.injections[cond.Variable] = append(cp.injections[cond.Variable], injection)
		}
	}
}

func (cp *ConfigParser) processBlock(condition string, content []string) {
	if condition == "otherwise" {
		return
	}

	parts := strings.SplitN(condition, "=", 2)
	if len(parts) != 2 {
		return
	}

	varName := parts[0]
	expectedVal := parts[1]

	for _, line := range content {
		peerdirs := strings.Split(line, " ")
		peerdirs = filterEmptyStrings(peerdirs)
		if len(peerdirs) > 0 {
			injection := &ConfigPEERDIRInjection{
				Variable:    varName,
				ExpectedVal: expectedVal,
				Peerdirs:    peerdirs,
			}
			cp.injections[varName] = append(cp.injections[varName], injection)
		}
	}
}

func (cp *ConfigParser) addGlobalInjection(peerdir string) {
	injection := &ConfigPEERDIRInjection{
		Variable:    "",
		ExpectedVal: "",
		Peerdirs:    []string{peerdir},
	}
	cp.injections["*"] = append(cp.injections["*"], injection)
}

func (cp *ConfigParser) GetInjections(variableName, currentValue string) []string {
	var result []string

	if injections, ok := cp.injections[variableName]; ok {
		for _, inj := range injections {
			if inj.ExpectedVal == currentValue {
				result = append(result, inj.Peerdirs...)
			}
		}
	}

	if injections, ok := cp.injections["*"]; ok {
		for _, inj := range injections {
			result = append(result, inj.Peerdirs...)
		}
	}

	return result
}

func (cp *ConfigParser) SetVariable(name, value string) {
	cp.variables[name] = value
}

func (cp *ConfigParser) GetVariable(name string) (string, bool) {
	val, ok := cp.variables[name]
	return val, ok
}

func (cp *ConfigParser) ResolveConfigPEERDIRS(variables map[string]string) []string {
	var result []string

	for varName, injections := range cp.injections {
		if varName == "*" {
			for _, inj := range injections {
				result = append(result, inj.Peerdirs...)
			}
			continue
		}

		if currentValue, ok := variables[varName]; ok {
			for _, inj := range injections {
				if inj.ExpectedVal == currentValue {
					result = append(result, inj.Peerdirs...)
				}
			}
		}
	}

	return result
}

func filterEmptyStrings(items []string) []string {
	var result []string
	for _, s := range items {
		if s != "" && !strings.HasPrefix(s, "#") {
			result = append(result, s)
		}
	}
	return result
}

func FindConfigFile(sourceRoot string) string {
	configPath := sourceRoot + "/build/ymake.core.conf"
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	return ""
}

func ReadConfigFile(configPath string) (map[string]string, map[string][]string, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("config file not found: %s", configPath)
	}

	file := Throw2(os.Open(configPath))
	defer file.Close()

	variables := make(map[string]string)
	peerdirInjections := make(map[string][]string)

	whenPattern := regexp.MustCompile(`^when\s*\(\$(\w+)\s*==\s*\"([^"]+)\"\)\s*\{`)
	peerdirPattern := regexp.MustCompile(`^\s*PEERDIR\s*\+=(.+)`)

	scanner := bufio.NewScanner(file)
	inWhenBlock := false
	currentVar := ""
	currentVal := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if whenMatch := whenPattern.FindStringSubmatch(line); len(whenMatch) > 0 {
			inWhenBlock = true
			currentVar = whenMatch[1]
			currentVal = whenMatch[2]
			continue
		}

		if inWhenBlock && strings.Contains(line, "}") {
			inWhenBlock = false
			currentVar = ""
			currentVal = ""
			continue
		}

		if inWhenBlock {
			if match := peerdirPattern.FindStringSubmatch(line); len(match) > 0 {
				peerdirs := strings.Split(strings.TrimSpace(match[1]), " ")
				peerdirs = filterEmptyStrings(peerdirs)
				key := currentVar + "=" + currentVal
				peerdirInjections[key] = append(peerdirInjections[key], peerdirs...)
			}
			continue
		}
	}

	return variables, peerdirInjections, nil
}
