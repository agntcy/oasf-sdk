// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

// MCPServer represents an MCP server configuration for GitHub Copilot.
type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// MCPInput represents an input configuration for GitHub Copilot MCP.
type MCPInput struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Password    bool   `json:"password"`
	Description string `json:"description"`
}

// GHCopilotMCPConfig represents GitHub Copilot MCP configuration.
type GHCopilotMCPConfig struct {
	Servers map[string]MCPServer `json:"servers"`
	Inputs  []MCPInput           `json:"inputs"`
}
