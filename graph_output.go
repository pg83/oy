package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
)

type GraphOutput struct {
	Conf   *Conf             `json:"conf"`
	Graph  []*GraphNode      `json:"graph"`
	Inputs GraphOutputInputs `json:"inputs"`
	Result []string          `json:"result"`
}

type GraphOutputInputs map[string]string

func (g GraphOutputInputs) MarshalJSON() ([]byte, error) {
	if g == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]string(g))
}

type Conf struct {
	Cache                     bool                   `json:"cache"`
	DefaultNodeRequirements   map[string]interface{} `json:"default_node_requirements"`
	Description               *Description           `json:"description"`
	ExecutionCost             *ExecutionCost         `json:"execution_cost"`
	ExplicitRemoteStoreUpload bool                   `json:"explicit_remote_store_upload"`
	GraphSize                 int                    `json:"graph_size"`
	Gsid                      string                 `json:"gsid"`
	Keepon                    bool                   `json:"keepon"`
	MinReqsErrors             int                    `json:"min_reqs_errors"`
	Platform                  string                 `json:"platform"`
	Resources                 []Resource             `json:"resources"`
}

type Description struct {
	Host     string `json:"host"`
	Platform string `json:"platform"`
	User     string `json:"user"`
}

type ExecutionCost struct {
	CPU              int `json:"cpu"`
	EvaluationErrors int `json:"evaluation_errors"`
}

type Resource struct {
	Name      string             `json:"name,omitempty"`
	Pattern   string             `json:"pattern"`
	Resource  string             `json:"resource,omitempty"`
	Resources []ResourcePlatform `json:"resources,omitempty"`
}

type ResourcePlatform struct {
	Platform string `json:"platform"`
	Resource string `json:"resource"`
}

func WriteGraphToFile(graph *Graph, outputPath string) {
	output := graph.ToOutput()

	file := Throw2(os.Create(outputPath))
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	Throw(encoder.Encode(output))
}

func WriteGraphToStdout(graph *Graph) {
	output := graph.ToOutput()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	Throw(encoder.Encode(output))
}

func generateSessionID() string {
	b := make([]byte, 16)
	Throw2(rand.Read(b))
	return hex.EncodeToString(b)
}
