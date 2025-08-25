# Translation SDK

## Prerequisites

- Translation SDK binary, distributed via [GitHub Releases](https://github.com/agntcy/oasf-sdk/releases)
- Translation SDK docker images, distributed via
[GitHub Packages](https://github.com/orgs/agntcy/packages?repo_name=oasf-sdk)

To start you need to have a [valid OASF data model](https://schema.oasf.outshift.com/0.5.0/objects/record) that you can
convert to different formats, let's save the following example manifest to a file named `model.json`:

```bash
cat << 'EOF' > model.json
{
  "record": {
    "name": "poc/integrations-agent-example",
    "version": "v1.0.0",
    "description": "An example agent with IDE integrations support",
    "authors": [
      "Adam Tagscherer <atagsche@cisco.com>"
    ],
    "created_at": "2025-06-16T17:06:37Z",
    "skills": [
      {
        "name": "schema.oasf.agntcy.org/skills/contextual_comprehension",
        "id": 10101
      }
    ],
    "locators": [
      {
        "type": "docker-image",
        "url": "https://ghcr.io/agntcy/dir/integrations-agent-example"
      }
    ],
    "extensions": [
      {
        "name": "schema.oasf.agntcy.org/features/runtime/mcp",
        "version": "v1.0.0",
        "data": {
          "servers": {
            "github": {
              "command": "docker",
              "args": [
                "run",
                "-i",
                "--rm",
                "-e",
                "GITHUB_PERSONAL_ACCESS_TOKEN",
                "ghcr.io/github/github-mcp-server"
              ],
              "env": {
                "GITHUB_PERSONAL_ACCESS_TOKEN": "${input:GITHUB_PERSONAL_ACCESS_TOKEN}"
              }
            }
          }
        }
      },
      {
        "name": "schema.oasf.agntcy.org/features/runtime/a2a",
        "version": "v1.0.0",
        "data": {
          "name": "example-agent",
          "description": "An agent that performs web searches and extracts information.",
          "url": "http://localhost:8000",
          "capabilities": {
            "streaming": true,
            "pushNotifications": false
          },
          "defaultInputModes": [
            "text"
          ],
          "defaultOutputModes": [
            "text"
          ],
          "skills": [
            {
              "id": "browser",
              "name": "browser automation",
              "description": "Performs web searches to retrieve information."
            }
          ]
        }
      }
    ],
    "signature": {}
  }
}
EOF
```

Now let's start the translation SDK server as a docker container, which will listen for incoming requests on port
`31234`:

```bash
docker run -p 31234:31234 ghcr.io/agntcy/oasf-sdk:latest
```

## VSCode MCP Config

Create a VSCode MCP Config from the OASF data model using the `RecordToVSCodeCopilot` RPC method.
You can pipe the output to a file wherever you want to save the MCP config.

```bash
grpcurl -plaintext \
  -d @ \
  localhost:31234 \
  translation.v1.TranslationService/RecordToVSCodeCopilot \
  <model.json \
  | jq
```

Output:
```json
{
  "data": {
    "mcpConfig": {
      "inputs": [
        {
          "description": "Secret value for GITHUB_PERSONAL_ACCESS_TOKEN",
          "id": "GITHUB_PERSONAL_ACCESS_TOKEN",
          "password": true,
          "type": "promptString"
        }
      ],
      "servers": {
        "github": {
          "args": [
            "run",
            "-i",
            "--rm",
            "-e",
            "GITHUB_PERSONAL_ACCESS_TOKEN",
            "ghcr.io/github/github-mcp-server"
          ],
          "command": "docker",
          "env": {
            "GITHUB_PERSONAL_ACCESS_TOKEN": "${input:GITHUB_PERSONAL_ACCESS_TOKEN}"
          }
        }
      }
    }
  }
}
```

## A2A Card extraction

To extract A2A card from the OASF data model, use the `RecordToA2ACard` RPC method.

```bash
grpcurl -plaintext \
  -d @ \
  localhost:31234 \
  translation.v1.TranslationService/RecordToA2A \
  <model.json \
  | jq
```

Output:
```json
{
  "data": {
    "a2aCard": {
      "capabilities": {
        "pushNotifications": false,
        "streaming": true
      },
      "defaultInputModes": [
        "text"
      ],
      "defaultOutputModes": [
        "text"
      ],
      "description": "An agent that performs web searches and extracts information.",
      "name": "example-agent",
      "skills": [
        {
          "description": "Performs web searches to retrieve information.",
          "id": "browser",
          "name": "browser automation"
        }
      ],
      "url": "http://localhost:8000"
    }
  }
}
```
