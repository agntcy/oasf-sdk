// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type MCPInput struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Password    bool   `json:"password"`
	Description string `json:"description"`
}

type GHCopilotMCPConfig struct {
	Servers map[string]MCPServer `json:"servers"`
	Inputs  []MCPInput           `json:"inputs"`
}

type A2ASkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type A2ACard struct {
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	URL                string          `json:"url"`
	Capabilities       map[string]bool `json:"capabilities"`
	DefaultInputModes  []string        `json:"defaultInputModes"`
	DefaultOutputModes []string        `json:"defaultOutputModes"`
	Skills             []A2ASkill      `json:"skills"`
}
