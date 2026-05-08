package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	REFERENCE_GRAPH_PATH = "/home/pg/monorepo/yatool_orig/sg.json"
	REFERENCE_TARGET     = "tools/archiver"
	EXPECTED_NODE_COUNT  = 3730
)

type GraphComparisonOptions struct {
	MaxMismatches int
}

type GraphComparisonResult struct {
	Mismatches      []string
	TotalMismatches int
}

func (r GraphComparisonResult) Err() error {
	if r.TotalMismatches == 0 {
		return nil
	}

	shown := len(r.Mismatches)
	lines := []string{fmt.Sprintf("graph mismatch: %d differences, showing first %d", r.TotalMismatches, shown)}
	lines = append(lines, r.Mismatches...)

	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}

type ValidationGraph struct {
	Conf   *ValidationConf   `json:"conf"`
	Graph  []ValidationNode  `json:"graph"`
	Inputs GraphInputs       `json:"inputs"`
	Result []string          `json:"result"`
	Nodes  []ValidationNode  `json:"-"`
	Index  map[string]int    `json:"-"`
	UIDSet map[string]string `json:"-"`
}

type GraphInputs struct {
	Values []string
}

func (gi *GraphInputs) UnmarshalJSON(data []byte) error {
	var arrayValues []string
	if err := json.Unmarshal(data, &arrayValues); err == nil {
		gi.Values = sortedStringsCopy(arrayValues)
		return nil
	}

	var objectValues map[string]string
	if err := json.Unmarshal(data, &objectValues); err == nil {
		values := make([]string, 0, len(objectValues)*2)

		for key, value := range objectValues {
			if value == "" || value == key {
				values = append(values, key)
				continue
			}

			values = append(values, key+"="+value)
		}

		gi.Values = sortedStringsCopy(values)
		return nil
	}

	return fmt.Errorf("inputs must be array or object")
}

func (gi GraphInputs) MarshalJSON() ([]byte, error) {
	return json.Marshal(gi.Values)
}

type ValidationConf struct {
	Cache                     bool            `json:"cache"`
	DefaultNodeRequirements   json.RawMessage `json:"default_node_requirements"`
	Description               json.RawMessage `json:"description"`
	ExecutionCost             json.RawMessage `json:"execution_cost"`
	ExplicitRemoteStoreUpload bool            `json:"explicit_remote_store_upload"`
	GraphSize                 int             `json:"graph_size"`
	Gsid                      string          `json:"gsid"`
	Keepon                    bool            `json:"keepon"`
	MinReqsErrors             int             `json:"min_reqs_errors"`
	Platform                  string          `json:"platform"`
	Resources                 json.RawMessage `json:"resources"`
}

type ValidationNode struct {
	UID              string                     `json:"uid"`
	SelfUID          string                     `json:"self_uid"`
	StatsUID         string                     `json:"stats_uid"`
	Cmds             []ValidationCommand        `json:"cmds"`
	Inputs           []string                   `json:"inputs"`
	Outputs          []string                   `json:"outputs"`
	Deps             []string                   `json:"deps"`
	KV               map[string]string          `json:"kv"`
	TargetProperties ValidationTargetProperties `json:"target_properties"`
	Env              map[string]string          `json:"env"`
	Platform         string                     `json:"platform"`
	Requirements     ValidationRequirements     `json:"requirements"`
	Sandboxing       bool                       `json:"sandboxing"`
	Tags             []string                   `json:"tags"`
	ForeignDeps      map[string][]string        `json:"foreign_deps"`
	HostPlatform     bool                       `json:"host_platform"`
	Extra            map[string]json.RawMessage `json:"-"`
	identityKey      string                     `json:"-"`
	identitySummary  validationNodeIdentity     `json:"-"`
}

type ValidationCommand struct {
	CmdArgs []string          `json:"cmd_args"`
	Env     map[string]string `json:"env"`
	Cwd     string            `json:"cwd"`
}

type ValidationTargetProperties struct {
	ModuleDir  string `json:"module_dir"`
	ModuleLang string `json:"module_lang"`
	ModuleType string `json:"module_type"`
}

type ValidationRequirements struct {
	CPU     int    `json:"cpu"`
	Network string `json:"network"`
	RAM     int    `json:"ram"`
}

type GraphValidator struct {
	referencePath string
	generated     *Graph
}

func NewGraphValidator(referencePath string, generated *Graph) *GraphValidator {
	return &GraphValidator{referencePath: referencePath, generated: generated}
}

func (gv *GraphValidator) Validate() error {
	generated := graphToValidationGraph(gv.generated)
	result := CompareGraphs(loadValidationGraph(gv.referencePath), generated, GraphComparisonOptions{})

	return result.Err()
}

func CompareGraphFiles(referencePath, generatedPath string, options GraphComparisonOptions) GraphComparisonResult {
	reference := loadValidationGraph(referencePath)
	generated := loadValidationGraph(generatedPath)

	return CompareGraphs(reference, generated, options)
}

func CompareGraphs(reference, generated *ValidationGraph, options GraphComparisonOptions) GraphComparisonResult {
	comparison := newGraphComparison(reference, generated, options)
	comparison.compare()

	return comparison.result
}

func loadValidationGraph(path string) *ValidationGraph {
	data := Throw2(os.ReadFile(path))
	var graph ValidationGraph
	Throw(json.Unmarshal(data, &graph))
	graph.finalize()

	return &graph
}

func graphToValidationGraph(graph *Graph) *ValidationGraph {
	output := graph.ToOutput()
	data := Throw2(json.Marshal(output))
	var validationGraph ValidationGraph
	Throw(json.Unmarshal(data, &validationGraph))
	validationGraph.finalize()

	return &validationGraph
}

func (g *ValidationGraph) finalize() {
	g.Nodes = g.Graph
	g.Index = make(map[string]int, len(g.Nodes))
	g.UIDSet = make(map[string]string, len(g.Nodes)*3)

	for i := range g.Nodes {
		node := &g.Nodes[i]
		node.Inputs = sortedStringsCopy(node.Inputs)
		node.Outputs = sortedStringsCopy(node.Outputs)
		node.Deps = sortedStringsCopy(node.Deps)
		node.Tags = sortedStringsCopy(node.Tags)
		node.ForeignDeps = normalizeForeignDeps(node.ForeignDeps)
		node.identitySummary = nodeIdentity(*node, nil)
		node.identityKey = canonicalJSON(node.identitySummary)

		if node.UID != "" {
			g.Index[node.UID] = i
			g.UIDSet[node.UID] = node.UID
		}

		if node.SelfUID != "" {
			g.UIDSet[node.SelfUID] = node.SelfUID
		}

		if node.StatsUID != "" {
			g.UIDSet[node.StatsUID] = node.StatsUID
		}
	}
}

type graphComparison struct {
	reference      *ValidationGraph
	generated      *ValidationGraph
	maxMismatches  int
	result         GraphComparisonResult
	refToGenUID    map[string]string
	genToRefUID    map[string]string
	refCanonical   map[string]string
	genCanonical   map[string]string
	refNodeColors  []string
	genNodeColors  []string
	refNodeKey     []string
	genNodeKey     []string
	unmappedReason []string
}

func newGraphComparison(reference, generated *ValidationGraph, options GraphComparisonOptions) *graphComparison {
	maxMismatches := options.MaxMismatches
	if maxMismatches <= 0 {
		maxMismatches = 20
	}

	return &graphComparison{
		reference:     reference,
		generated:     generated,
		maxMismatches: maxMismatches,
		refToGenUID:   make(map[string]string),
		genToRefUID:   make(map[string]string),
		refCanonical:  make(map[string]string),
		genCanonical:  make(map[string]string),
	}
}

func (c *graphComparison) compare() {
	c.buildUIDMapping()
	c.compareGraphShape()
	c.compareConf()
	c.compareTopLevelInputs()
	c.compareResults()
	c.compareNodes()
}

func (c *graphComparison) addMismatch(format string, args ...any) {
	c.result.TotalMismatches++

	if len(c.result.Mismatches) < c.maxMismatches {
		c.result.Mismatches = append(c.result.Mismatches, fmt.Sprintf(format, args...))
	}
}

func (c *graphComparison) compareGraphShape() {
	if len(c.reference.Nodes) != len(c.generated.Nodes) {
		c.addMismatch("graph node count: ref=%d gen=%d", len(c.reference.Nodes), len(c.generated.Nodes))
	}

	refGraphSize := 0
	genGraphSize := 0
	if c.reference.Conf != nil {
		refGraphSize = c.reference.Conf.GraphSize
	}
	if c.generated.Conf != nil {
		genGraphSize = c.generated.Conf.GraphSize
	}
	if refGraphSize != genGraphSize {
		c.addMismatch("conf.graph_size: ref=%d gen=%d", refGraphSize, genGraphSize)
	}
}

func (c *graphComparison) compareConf() {
	if c.reference.Conf == nil || c.generated.Conf == nil {
		if c.reference.Conf != c.generated.Conf {
			c.addMismatch("conf presence: ref=%t gen=%t", c.reference.Conf != nil, c.generated.Conf != nil)
		}

		return
	}

	ref := c.reference.Conf
	gen := c.generated.Conf
	c.compareBool("conf.cache", ref.Cache, gen.Cache)
	c.compareRawJSON("conf.default_node_requirements", ref.DefaultNodeRequirements, gen.DefaultNodeRequirements)
	c.compareRawJSON("conf.execution_cost", ref.ExecutionCost, gen.ExecutionCost)
	c.compareBool("conf.explicit_remote_store_upload", ref.ExplicitRemoteStoreUpload, gen.ExplicitRemoteStoreUpload)
	c.compareInt("conf.graph_size", ref.GraphSize, gen.GraphSize)
	c.compareBool("conf.keepon", ref.Keepon, gen.Keepon)
	c.compareInt("conf.min_reqs_errors", ref.MinReqsErrors, gen.MinReqsErrors)
	c.compareString("conf.platform", ref.Platform, gen.Platform)
	c.compareRawJSON("conf.resources", ref.Resources, gen.Resources)
}

func (c *graphComparison) compareTopLevelInputs() {
	c.compareStringSlices("inputs", c.reference.Inputs.Values, c.generated.Inputs.Values)
}

func (c *graphComparison) compareResults() {
	if len(c.reference.Result) != len(c.generated.Result) {
		c.addMismatch("result cardinality: ref=%d gen=%d", len(c.reference.Result), len(c.generated.Result))
		return
	}

	expected := make([]string, 0, len(c.reference.Result))
	for _, uid := range c.reference.Result {
		mapped, ok := c.refToGenUID[uid]
		if !ok {
			c.addMismatch("result uid %s has no generated mapping", uid)
			continue
		}

		expected = append(expected, mapped)
	}

	c.compareStringSlices("result", sortedStringsCopy(expected), sortedStringsCopy(c.generated.Result))
}

func (c *graphComparison) compareNodes() {
	for i := range c.reference.Nodes {
		refNode := c.reference.Nodes[i]
		genUID, ok := c.refToGenUID[refNode.UID]
		if !ok {
			if i < len(c.unmappedReason) && c.unmappedReason[i] != "" {
				c.addMismatch("missing generated node for ref_uid=%s module_dir=%s identity=%s: %s", refNode.UID, refNode.TargetProperties.ModuleDir, summarizeIdentity(refNode.identityKey), c.unmappedReason[i])
			} else {
				c.addMismatch("missing generated node for ref_uid=%s module_dir=%s identity=%s", refNode.UID, refNode.TargetProperties.ModuleDir, summarizeIdentity(refNode.identityKey))
			}
			continue
		}

		genIndex, ok := c.generated.Index[genUID]
		if !ok {
			c.addMismatch("mapped generated uid missing from index: ref_uid=%s gen_uid=%s", refNode.UID, genUID)
			continue
		}

		genNode := c.generated.Nodes[genIndex]
		c.compareNode(refNode, genNode)
	}

	for _, genNode := range c.generated.Nodes {
		if _, ok := c.genToRefUID[genNode.UID]; !ok {
			c.addMismatch("extra generated node gen_uid=%s module_dir=%s identity=%s", genNode.UID, genNode.TargetProperties.ModuleDir, summarizeIdentity(genNode.identityKey))
		}
	}
}

func (c *graphComparison) compareNode(refNode, genNode ValidationNode) {
	context := nodeContext(refNode, genNode)
	c.compareString(context+" field=target_properties.module_dir", refNode.TargetProperties.ModuleDir, genNode.TargetProperties.ModuleDir)
	c.compareString(context+" field=target_properties.module_lang", refNode.TargetProperties.ModuleLang, genNode.TargetProperties.ModuleLang)
	c.compareString(context+" field=target_properties.module_type", refNode.TargetProperties.ModuleType, genNode.TargetProperties.ModuleType)
	c.compareString(context+" field=platform", refNode.Platform, genNode.Platform)
	c.compareString(context+" field=host_platform", fmt.Sprint(refNode.HostPlatform), fmt.Sprint(genNode.HostPlatform))
	c.compareRawJSON(context+" field=requirements", canonicalRaw(refNode.Requirements), canonicalRaw(genNode.Requirements))
	c.compareBool(context+" field=sandboxing", refNode.Sandboxing, genNode.Sandboxing)
	c.compareStringSlices(context+" field=tags", sortedStringsCopy(refNode.Tags), sortedStringsCopy(genNode.Tags))
	c.compareRawJSON(context+" field=foreign_deps", canonicalRaw(normalizeForeignDeps(refNode.ForeignDeps)), canonicalRaw(normalizeForeignDeps(genNode.ForeignDeps)))
	c.compareStringSlices(context+" field=inputs", sortedStringsCopy(refNode.Inputs), sortedStringsCopy(genNode.Inputs))
	c.compareStringSlices(context+" field=outputs", sortedStringsCopy(refNode.Outputs), sortedStringsCopy(genNode.Outputs))
	c.compareStringMap(context+" field=env", refNode.Env, genNode.Env)
	c.compareStringMap(context+" field=kv", c.normalizeKV(refNode.KV, c.refCanonical), c.normalizeKV(genNode.KV, c.genCanonical))
	c.compareCommands(context, refNode.Cmds, genNode.Cmds)
	c.compareStringSlices(context+" field=deps", c.mappedReferenceDeps(refNode), sortedStringsCopy(genNode.Deps))
}

func (c *graphComparison) compareCommands(context string, refCmds, genCmds []ValidationCommand) {
	if len(refCmds) != len(genCmds) {
		c.addMismatch("%s field=cmds length: ref=%d gen=%d", context, len(refCmds), len(genCmds))
		return
	}

	for i := range refCmds {
		prefix := fmt.Sprintf("%s field=cmds[%d]", context, i)
		c.compareStringSlices(prefix+".cmd_args", refCmds[i].CmdArgs, genCmds[i].CmdArgs)
		c.compareStringMap(prefix+".env", refCmds[i].Env, genCmds[i].Env)
		c.compareString(prefix+".cwd", refCmds[i].Cwd, genCmds[i].Cwd)
	}
}

func (c *graphComparison) mappedReferenceDeps(node ValidationNode) []string {
	mapped := make([]string, 0, len(node.Deps))

	for _, dep := range node.Deps {
		genUID, ok := c.refToGenUID[dep]
		if !ok {
			mapped = append(mapped, "<unmapped-ref-uid:"+dep+">")
			continue
		}

		mapped = append(mapped, genUID)
	}

	return sortedStringsCopy(mapped)
}

func (c *graphComparison) buildUIDMapping() {
	if len(c.reference.Nodes) != len(c.generated.Nodes) {
		c.mapUniqueLooseKeys()
		return
	}

	refColors := initialColors(c.reference.Nodes)
	genColors := initialColors(c.generated.Nodes)
	maxIterations := max(len(c.reference.Nodes), len(c.generated.Nodes)) + 1

	for i := 0; i < maxIterations; i++ {
		nextRef := refineColors(c.reference, refColors)
		nextGen := refineColors(c.generated, genColors)

		if equalStringSlices(nextRef, refColors) && equalStringSlices(nextGen, genColors) {
			break
		}

		refColors = nextRef
		genColors = nextGen
	}

	c.refNodeColors = refColors
	c.genNodeColors = genColors
	c.refNodeKey = refColors
	c.genNodeKey = genColors
	c.unmappedReason = make([]string, len(c.reference.Nodes))

	refGroups := groupNodeIndexes(refColors)
	genGroups := groupNodeIndexes(genColors)

	for color, refIndexes := range refGroups {
		genIndexes := genGroups[color]
		if len(refIndexes) != len(genIndexes) {
			for _, refIndex := range refIndexes {
				c.unmappedReason[refIndex] = fmt.Sprintf("color cardinality mismatch ref=%d gen=%d", len(refIndexes), len(genIndexes))
			}

			continue
		}

		sortNodeIndexes(c.reference.Nodes, refIndexes, refColors)
		sortNodeIndexes(c.generated.Nodes, genIndexes, genColors)

		for i, refIndex := range refIndexes {
			genIndex := genIndexes[i]
			refUID := c.reference.Nodes[refIndex].UID
			genUID := c.generated.Nodes[genIndex].UID
			c.refToGenUID[refUID] = genUID
			c.genToRefUID[genUID] = refUID
			canonical := fmt.Sprintf("N%06d", len(c.refCanonical))
			c.refCanonical[refUID] = canonical
			c.genCanonical[genUID] = canonical

			if c.reference.Nodes[refIndex].SelfUID != "" {
				c.refCanonical[c.reference.Nodes[refIndex].SelfUID] = canonical
			}
			if c.reference.Nodes[refIndex].StatsUID != "" {
				c.refCanonical[c.reference.Nodes[refIndex].StatsUID] = canonical
			}
			if c.generated.Nodes[genIndex].SelfUID != "" {
				c.genCanonical[c.generated.Nodes[genIndex].SelfUID] = canonical
			}
			if c.generated.Nodes[genIndex].StatsUID != "" {
				c.genCanonical[c.generated.Nodes[genIndex].StatsUID] = canonical
			}
		}
	}

	c.mapUniqueLooseKeys()
}

func (c *graphComparison) mapUniqueLooseKeys() {
	if c.unmappedReason == nil {
		c.unmappedReason = make([]string, len(c.reference.Nodes))
	}

	refGroups := make(map[string][]int)
	genGroups := make(map[string][]int)

	for i, node := range c.reference.Nodes {
		if _, ok := c.refToGenUID[node.UID]; ok {
			continue
		}

		refGroups[nodeLooseKey(node)] = append(refGroups[nodeLooseKey(node)], i)
	}

	for i, node := range c.generated.Nodes {
		if _, ok := c.genToRefUID[node.UID]; ok {
			continue
		}

		genGroups[nodeLooseKey(node)] = append(genGroups[nodeLooseKey(node)], i)
	}

	for key, refIndexes := range refGroups {
		genIndexes := genGroups[key]
		if len(refIndexes) != 1 || len(genIndexes) != 1 {
			continue
		}

		refIndex := refIndexes[0]
		genIndex := genIndexes[0]
		refUID := c.reference.Nodes[refIndex].UID
		genUID := c.generated.Nodes[genIndex].UID
		c.refToGenUID[refUID] = genUID
		c.genToRefUID[genUID] = refUID
		canonical := fmt.Sprintf("N%06d", len(c.refCanonical))
		c.refCanonical[refUID] = canonical
		c.genCanonical[genUID] = canonical
		c.unmappedReason[refIndex] = ""

		if c.reference.Nodes[refIndex].SelfUID != "" {
			c.refCanonical[c.reference.Nodes[refIndex].SelfUID] = canonical
		}
		if c.reference.Nodes[refIndex].StatsUID != "" {
			c.refCanonical[c.reference.Nodes[refIndex].StatsUID] = canonical
		}
		if c.generated.Nodes[genIndex].SelfUID != "" {
			c.genCanonical[c.generated.Nodes[genIndex].SelfUID] = canonical
		}
		if c.generated.Nodes[genIndex].StatsUID != "" {
			c.genCanonical[c.generated.Nodes[genIndex].StatsUID] = canonical
		}
	}
}

func initialColors(nodes []ValidationNode) []string {
	colors := make([]string, len(nodes))

	for i := range nodes {
		colors[i] = nodes[i].identityKey
	}

	return colors
}

func refineColors(graph *ValidationGraph, colors []string) []string {
	next := make([]string, len(graph.Nodes))

	for i, node := range graph.Nodes {
		depColors := make([]string, 0, len(node.Deps))

		for _, dep := range node.Deps {
			if depIndex, ok := graph.Index[dep]; ok {
				depColors = append(depColors, colors[depIndex])
			} else {
				depColors = append(depColors, "<missing:"+dep+">")
			}
		}

		sort.Strings(depColors)
		next[i] = canonicalJSON(map[string]any{"identity": node.identityKey, "deps": depColors})
	}

	return next
}

func groupNodeIndexes(colors []string) map[string][]int {
	groups := make(map[string][]int)

	for i, color := range colors {
		groups[color] = append(groups[color], i)
	}

	return groups
}

func sortNodeIndexes(nodes []ValidationNode, indexes []int, colors []string) {
	sort.Slice(indexes, func(i, j int) bool {
		left := nodeStableFingerprint(nodes[indexes[i]], colors, indexes[i])
		right := nodeStableFingerprint(nodes[indexes[j]], colors, indexes[j])

		return left < right
	})
}

func nodeStableFingerprint(node ValidationNode, colors []string, index int) string {
	return canonicalJSON(map[string]any{
		"identity": node.identityKey,
		"deps":     sortedStringsCopy(node.Deps),
		"index":    index,
	})
}

type validationNodeIdentity struct {
	TargetProperties ValidationTargetProperties `json:"target_properties"`
	Platform         string                     `json:"platform"`
	HostPlatform     bool                       `json:"host_platform"`
	Requirements     ValidationRequirements     `json:"requirements"`
	Sandboxing       bool                       `json:"sandboxing"`
	Tags             []string                   `json:"tags"`
	ForeignDeps      map[string][]string        `json:"foreign_deps"`
	Inputs           []string                   `json:"inputs"`
	Outputs          []string                   `json:"outputs"`
	Cmds             []ValidationCommand        `json:"cmds"`
	Env              map[string]string          `json:"env"`
	KV               map[string]string          `json:"kv"`
}

func nodeIdentity(node ValidationNode, canonical map[string]string) validationNodeIdentity {
	return validationNodeIdentity{
		TargetProperties: node.TargetProperties,
		Platform:         node.Platform,
		HostPlatform:     node.HostPlatform,
		Requirements:     node.Requirements,
		Sandboxing:       node.Sandboxing,
		Tags:             sortedStringsCopy(node.Tags),
		ForeignDeps:      normalizeForeignDeps(node.ForeignDeps),
		Inputs:           sortedStringsCopy(node.Inputs),
		Outputs:          sortedStringsCopy(node.Outputs),
		Cmds:             node.Cmds,
		Env:              copyStringMap(node.Env),
		KV:               normalizeKVMap(node.KV, canonical),
	}
}

func (c *graphComparison) normalizeKV(kv map[string]string, canonical map[string]string) map[string]string {
	return normalizeKVMap(kv, canonical)
}

func normalizeKVMap(kv map[string]string, canonical map[string]string) map[string]string {
	if kv == nil {
		return nil
	}

	normalized := make(map[string]string, len(kv))

	for key, value := range kv {
		if isUIDKey(key) {
			if canonical == nil {
				normalized[key] = "$(UID)"
				continue
			}

			normalized[key] = normalizeUIDString(value, canonical)
			continue
		}

		if _, ok := canonical[value]; ok {
			normalized[key] = normalizeUIDString(value, canonical)
			continue
		}

		normalized[key] = normalizeUIDString(value, canonical)
	}

	return normalized
}

func nodeLooseKey(node ValidationNode) string {
	return canonicalJSON(map[string]any{
		"target_properties": node.TargetProperties,
		"platform":          node.Platform,
		"host_platform":     node.HostPlatform,
		"inputs":            sortedStringsCopy(node.Inputs),
		"outputs":           sortedStringsCopy(node.Outputs),
	})
}

func isUIDKey(key string) bool {
	switch strings.ToLower(key) {
	case "uid", "self_uid", "stats_uid":
		return true
	default:
		return strings.HasSuffix(strings.ToLower(key), "_uid")
	}
}

func normalizeUIDString(value string, canonical map[string]string) string {
	if canonical == nil || value == "" {
		return value
	}

	normalized := value
	keys := make([]string, 0, len(canonical))

	for uid := range canonical {
		keys = append(keys, uid)
	}

	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })

	for _, uid := range keys {
		if uid == "" || !strings.Contains(normalized, uid) {
			continue
		}

		normalized = strings.ReplaceAll(normalized, uid, "$(UID:"+canonical[uid]+")")
	}

	return normalized
}

func normalizeForeignDeps(foreignDeps map[string][]string) map[string][]string {
	if foreignDeps == nil {
		return nil
	}

	normalized := make(map[string][]string, len(foreignDeps))

	for platform, deps := range foreignDeps {
		normalized[platform] = sortedStringsCopy(deps)
	}

	return normalized
}

func (c *graphComparison) compareString(path, ref, gen string) {
	if ref != gen {
		c.addMismatch("%s: ref=%q gen=%q", path, ref, gen)
	}
}

func (c *graphComparison) compareBool(path string, ref, gen bool) {
	if ref != gen {
		c.addMismatch("%s: ref=%t gen=%t", path, ref, gen)
	}
}

func (c *graphComparison) compareInt(path string, ref, gen int) {
	if ref != gen {
		c.addMismatch("%s: ref=%d gen=%d", path, ref, gen)
	}
}

func (c *graphComparison) compareStringSlices(path string, ref, gen []string) {
	if len(ref) != len(gen) {
		c.addMismatch("%s length: ref=%d gen=%d ref_values=%v gen_values=%v", path, len(ref), len(gen), ref, gen)
		return
	}

	for i := range ref {
		if ref[i] != gen[i] {
			c.addMismatch("%s[%d]: ref=%q gen=%q", path, i, ref[i], gen[i])
		}
	}
}

func (c *graphComparison) compareStringMap(path string, ref, gen map[string]string) {
	if len(ref) != len(gen) {
		c.addMismatch("%s size: ref=%d gen=%d ref_values=%v gen_values=%v", path, len(ref), len(gen), ref, gen)
		return
	}

	keys := make([]string, 0, len(ref))
	for key := range ref {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		genValue, ok := gen[key]
		if !ok {
			c.addMismatch("%s[%q] missing in generated", path, key)
			continue
		}

		if ref[key] != genValue {
			c.addMismatch("%s[%q]: ref=%q gen=%q", path, key, ref[key], genValue)
		}
	}
}

func (c *graphComparison) compareRawJSON(path string, ref, gen json.RawMessage) {
	refCanonical := canonicalRawString(ref)
	genCanonical := canonicalRawString(gen)

	if refCanonical != genCanonical {
		c.addMismatch("%s: ref=%s gen=%s", path, refCanonical, genCanonical)
	}
}

func sortedStringsCopy(values []string) []string {
	if values == nil {
		return nil
	}

	copyValues := make([]string, len(values))
	copy(copyValues, values)
	sort.Strings(copyValues)

	return copyValues
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}

	copyValues := make(map[string]string, len(values))

	for key, value := range values {
		copyValues[key] = value
	}

	return copyValues
}

func canonicalJSON(value any) string {
	return string(Throw2(json.Marshal(value)))
}

func canonicalRaw(value any) json.RawMessage {
	return json.RawMessage(canonicalJSON(value))
}

func canonicalRawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "null"
	}

	var value any
	Throw(json.Unmarshal(raw, &value))

	return canonicalJSON(value)
}

func nodeContext(refNode, genNode ValidationNode) string {
	target := refNode.TargetProperties.ModuleDir
	if target == "" {
		target = genNode.TargetProperties.ModuleDir
	}

	return fmt.Sprintf("node ref_uid=%s gen_uid=%s target=%s", refNode.UID, genNode.UID, target)
}

func summarizeIdentity(identity string) string {
	if len(identity) <= 160 {
		return identity
	}

	return identity[:160] + "..."
}

func max(left, right int) int {
	if left > right {
		return left
	}

	return right
}
