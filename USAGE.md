# Translation Service

## Prerequisites

- Translation SDK binary, distributed via [GitHub Releases](https://github.com/agntcy/oasf-sdk/releases)
- Translation SDK docker images, distributed via
  [GitHub Packages](https://github.com/orgs/agntcy/packages?repo_name=oasf-sdk)
- Docker

Let's start the OASF SDK as a docker container, which will listen for incoming requests on port `31234`:

```bash
docker run -p 31234:31234 ghcr.io/agntcy/oasf-sdk:latest
```

Or, for local testing purposes:

```bash
task build
docker run -p 31234:31234 oasf-sdk:latest
```

## GitHub Copilot config

Create a GitHub Copilot config from the OASF data model using the `RecordToGHCopilot` RPC method.
You can pipe the output to a file wherever you want to save the config.

```bash
cat tests/fixtures/translation_0.8.0_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToGHCopilot
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

To extract A2A card from the OASF data model, use the `RecordToA2A` RPC method.

```bash
cat tests/fixtures/translation_0.8.0_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToA2A
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
      "defaultInputModes": ["text"],
      "defaultOutputModes": ["text"],
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

## MCP Registry to OASF Record

To convert an MCP Registry server.json to an OASF record, use the `MCPToRecord` RPC method. This translates the deployment metadata from an MCP server.json file into an OASF 0.8.0 record with the MCP module populated.

**Note:** The MCP Registry server.json contains deployment metadata (packages, remotes, etc.), not runtime capabilities (tools, resources, prompts). Those are discovered via the MCP protocol when a client connects to the server.

```bash
cat tests/fixtures/translation_mcp.json | jq '{data: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/MCPToRecord
```

Output:

```json
{
  "record": {
    "name": "io.github.modelcontextprotocol/filesystem",
    "schema_version": "0.8.0",
    "version": "1.0.0",
    "description": "Secure file system operations through MCP",
    "authors": ["modelcontextprotocol"],
    "created_at": "2025-10-06T00:00:00Z",
    "skills": [
      {
        "id": 0,
        "name": "base_skill"
      }
    ],
    "locators": [
      {
        "type": "source_code",
        "url": "https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem"
      }
    ],
    "domains": [
      {
        "id": 0,
        "name": "base_domain"
      }
    ],
    "modules": [
      {
        "name": "integration/mcp",
        "data": {
          "servers": [
            {
              "name": "Filesystem",
              "type": "local",
              "capabilities": [],
              "command": "npx",
              "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
              "description": "Secure file system operations through MCP",
              "env_vars": [
                {
                  "name": "MCP_LOG_LEVEL",
                  "value": "info"
                }
              ]
            }
          ]
        }
      }
    ]
  }
}
```

## Agent Skills (SKILL.md)

The Agent Skills translator converts between SKILL.md files (used by Claude and other AI coding assistants) and OASF records.

### SKILL.md to OASF Record

To convert a SKILL.md file to an OASF record, wrap the file content in a `skillMarkdown` field and call `SkillMarkdownToRecord`:

```bash
jq -n --rawfile md tests/fixtures/translation_skill.json '{"data": {"skillMarkdown": $md}}' \
  | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/SkillMarkdownToRecord
```

Or using the fixture directly:

```bash
cat tests/fixtures/translation_skill.json | jq '{data: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/SkillMarkdownToRecord
```

Output:

```json
{
  "record": {
    "name": "pdf-processing",
    "schema_version": "1.0.0",
    "version": "1.0.0",
    "description": "Extract PDF text and merge files. Use when handling PDFs.",
    "authors": [],
    "created_at": "2025-01-01T00:00:00Z",
    "skills": [],
    "domains": [],
    "modules": [
      {
        "name": "agentskills",
        "data": {
          "skill_file": "SKILL.md",
          "skill_manifest": {
            "name": "pdf-processing",
            "description": "Extract PDF text and merge files. Use when handling PDFs.",
            "license": "Apache-2.0",
            "compatibility": "Requires python3",
            "version": "1.0.0",
            "allowed_tools": ["Read", "Bash(jq:*)"],
            "frontmatter_metadata": {
              "author": "example-org",
              "version": "1.0.0"
            }
          },
          "skill_body": "# PDF Processing Skill\n\nUse this skill when handling PDFs."
        }
      }
    ]
  }
}
```

### OASF Record to SKILL.md

To render a SKILL.md from an OASF record that contains an `agentskills` module, use `RecordToSkillMarkdown`:

```bash
cat tests/fixtures/translation_agentskills_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToSkillMarkdown
```

Output:

```json
{
  "data": "---\nname: pdf-processing\ndescription: Extract PDF text and merge files. Use when handling PDFs.\nlicense: Apache-2.0\ncompatibility: Requires python3\nversion: 1.0.0\nallowed-tools: Read Bash(jq:*)\nmetadata:\n  author: example-org\n  version: 1.0.0\n---\n"
}
```

## AI Catalog

Project an OASF record onto its AI Catalog entry using the `RecordToCatalog` RPC method. A single known integration module (`integration/mcp`, `integration/a2a`, `core/language_model/agentskills`) yields a leaf entry whose `media_type` matches the module and whose `data` is the module's structured data; multiple modules yield an `application/ai-catalog+json` container with one nested entry per module. A `cid` is required; `host` (defaults to `org.agntcy`) and `specVersion` (defaults to `1.0`) are optional.

```bash
cat tests/fixtures/translation_catalog_record.json | jq '{record: ., cid: "baeareibxiiy45pg4bjwhbijgh35epzjhnh6lvaxts2qggcgssn3glzdh64"}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToCatalog
```

Output:

```json
{
  "data": {
    "identifier": "urn:ai:org.agntcy:cid:baeareibxiiy45pg4bjwhbijgh35epzjhnh6lvaxts2qggcgssn3glzdh64",
    "media_type": "application/mcp-server+json",
    "display_name": "example-mcp-agent",
    "version": "1.0.0",
    "description": "Example agent exposing an MCP server.",
    "updated_at": "2026-01-01T00:00:00Z",
    "tags": [
      "oasf:v1.0.0:skills:natural_language_processing/natural_language_generation",
      "oasf:v1.0.0:domains:technology/software_engineering"
    ],
    "data": {
      "servers": [
        {
          "name": "example",
          "type": "local"
        }
      ]
    }
  }
}
```

# Schema Service

The OASF SDK Schema Service provides access to OASF schema definitions, allowing you to fetch schema content and extract specific sections like skills, domains, and modules from the schema.

## Golang example

```bash
go get github.com/agntcy/oasf-sdk/pkg@v0.0.9
```

Package based usage:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/agntcy/oasf-sdk/pkg/schema"
)

func main() {
	// Create a new schema instance with schema URL (cache disabled by default)
	s, err := schema.New("https://schema.oasf.outshift.com")
	if err != nil {
		log.Fatalf("Failed to create schema instance: %v", err)
	}

	// Optional: enable dynamic cache (only requested data is cached)
	cachedClient, err := schema.New("https://schema.oasf.outshift.com", schema.WithCache(true))
	if err != nil {
		log.Fatalf("Failed to create cached schema instance: %v", err)
	}
	_ = cachedClient

	ctx := context.Background()

	// Get available schema versions from the server
	versions, err := s.GetAvailableSchemaVersions(ctx)
	if err != nil {
		log.Fatalf("Failed to get available schema versions: %v", err)
	}
	fmt.Printf("Available schema versions: %v\n", versions)

	// Get the default schema version (cached after first fetch)
	defaultVersion, err := s.GetDefaultSchemaVersion(ctx)
	if err != nil {
		log.Fatalf("Failed to get default version: %v", err)
	}
	fmt.Printf("Default schema version: %s\n", defaultVersion)

	// Get full schema content for version 0.8.0 (using WithSchemaVersion option)
	schemaContent, err := s.GetRecordJSONSchema(ctx, schema.WithSchemaVersion("0.8.0"))
	if err != nil {
		log.Fatalf("Failed to get schema content: %v", err)
	}

	fmt.Printf("Schema version 0.8.0 loaded successfully (%d bytes)\n", len(schemaContent))

	// Get nested skill categories (using default version - no option needed)
	skillsData, err := s.GetSchemaSkills(ctx)
	if err != nil {
		log.Fatalf("Failed to get skills: %v", err)
	}
	fmt.Printf("Found %d top-level skill categories\n", len(skillsData))

	// Get nested domain categories (using specific version)
	domainsData, err := s.GetSchemaDomains(ctx, schema.WithSchemaVersion("0.8.0"))
	if err != nil {
		log.Fatalf("Failed to get domains: %v", err)
	}
	fmt.Printf("Found %d top-level domain categories\n", len(domainsData))

	// Get nested module categories (using default version)
	modulesData, err := s.GetSchemaModules(ctx)
	if err != nil {
		log.Fatalf("Failed to get modules: %v", err)
	}
	fmt.Printf("Found %d top-level module categories\n", len(modulesData))

	// Get a specific JSON schema by type and name (using WithSchemaVersion option)
	agentSkillsSchema, err := s.GetJSONSchema(ctx, schema.EntityTypeModules, "agentskills", schema.WithSchemaVersion("1.0.0"))
	if err != nil {
		log.Fatalf("Failed to get agent skills module schema: %v", err)
	}
	fmt.Printf("Agent skills module schema loaded (%d bytes)\n", len(agentSkillsSchema))

	agentSkillsManifestSchema, err := s.GetJSONSchema(ctx, schema.EntityTypeObjects, "agentskills_manifest", schema.WithSchemaVersion("1.0.0"))
	if err != nil {
		log.Fatalf("Failed to get agent skills manifest schema: %v", err)
	}
	fmt.Printf("Agent skills manifest schema loaded (%d bytes)\n", len(agentSkillsManifestSchema))
}
```

## Supported Schema Versions

The schema package supports the following versions:

- `0.7.0` - Uses `/schema/0.7.0/objects/record` endpoint
- `0.8.0` - Uses `/schema/0.8.0/objects/record` endpoint
- `1.0.0` - Uses `/schema/1.0.0/objects/record` endpoint

You can get the list of supported versions programmatically by fetching from the server:

```go
s, err := schema.New("https://schema.oasf.outshift.com")
if err != nil {
	log.Fatalf("Failed to create schema: %v", err)
}

versions, err := s.GetAvailableSchemaVersions(ctx)
if err != nil {
	log.Fatalf("Failed to get versions: %v", err)
}
// Returns: []string{"0.7.0", "0.8.0", "1.0.0", ...} (fetched from server)
```

## API Methods

### GetDefaultSchemaVersion

Returns the default schema version from the server. The version is cached after the first fetch:

```go
defaultVersion, err := s.GetDefaultSchemaVersion(ctx)
```

### Cache behavior

- Caching is disabled by default.
- Enable dynamic caching via constructor option: `schema.New(url, schema.WithCache(true))`.
- Dynamic caching stores only data that has been requested.
- Clear in-memory cache with `s.ClearCache()`.

### GetRecordJSONSchema

Fetches the complete JSON schema. If no version is provided via `WithSchemaVersion()`, the default version from the server is used:

```go
// Using default version
schemaContent, err := s.GetRecordJSONSchema(ctx)

// Using specific version
schemaContent, err := s.GetRecordJSONSchema(ctx, schema.WithSchemaVersion("0.8.0"))
```

### Convenience Methods

All convenience methods accept optional `WithSchemaVersion()` option. If omitted, the default version is used. These methods call the new taxonomy endpoints and return nested Go structs (`schema.Taxonomy`):

- `GetSchemaSkills(ctx, ...SchemaOption)` - calls `/api/<version>/skill_categories`
- `GetSchemaDomains(ctx, ...SchemaOption)` - calls `/api/<version>/domain_categories`
- `GetSchemaModules(ctx, ...SchemaOption)` - calls `/api/<version>/module_categories`

### Accessing Agent Skills data from a record

The translator package can convert between SKILL.md content and OASF records directly.

Convert a SKILL.md to a record:

```go
skillData, _ := structpb.NewStruct(map[string]any{
  "skillMarkdown": "---\nname: my-skill\ndescription: Does something.\n---\nBody here.\n",
})
record, err := translator.SkillMarkdownToRecord(skillData)
if err != nil {
  log.Fatalf("Failed to convert SKILL.md to record: %v", err)
}
```

`WithAuthors` and `WithRecordVersion` override source-derived values when provided. Otherwise, `*ToRecord` functions resolve authors from the source (e.g. SKILL.md `metadata.author`, A2A provider organization, MCP namespace vendor) and fall back to `["Unknown"]`; record `version` defaults to `v1.0.0` when absent from the source. `WithVersion` sets the OASF `schema_version` separately.

Render a SKILL.md from a record:

```go
skillMarkdown, err := translator.RecordToSkillMarkdown(record)
if err != nil {
  log.Fatalf("Failed to render SKILL.md from record: %v", err)
}
fmt.Printf("SKILL.md content:\n%s", skillMarkdown)
```

Example:

```go
// Using default version
skills, err := s.GetSchemaSkills(ctx)

// Using specific version
skills, err := s.GetSchemaSkills(ctx, schema.WithSchemaVersion("0.7.0"))
```

## gRPC API example

The SDK server also exposes the schema functionality via gRPC.

**Get default schema version:**

```bash
grpcurl -plaintext \
  -d '{"schema_url":"https://schema.oasf.outshift.com"}' \
  localhost:31234 \
  agntcy.oasfsdk.schema.v1.SchemaService/GetDefaultSchemaVersion
```

**Get available schema versions:**

```bash
grpcurl -plaintext \
  -d '{"schema_url":"https://schema.oasf.outshift.com"}' \
  localhost:31234 \
  agntcy.oasfsdk.schema.v1.SchemaService/GetAvailableSchemaVersions
```

**Get record schema (specific version):**

```bash
grpcurl -plaintext \
  -d '{"schema_url":"https://schema.oasf.outshift.com","schema_version":"0.8.0"}' \
  localhost:31234 \
  agntcy.oasfsdk.schema.v1.SchemaService/GetRecordJSONSchema
```

**Get a JSON schema by full URL:**

```bash
grpcurl -plaintext \
  -d '{"url":"https://schema.oasf.outshift.com/schema/1.0.0/skills/contextual_comprehension"}' \
  localhost:31234 \
  agntcy.oasfsdk.schema.v1.SchemaService/GetJSONSchema
```

**Get nested skill/domain/module categories:**

```bash
grpcurl -plaintext \
  -d '{"schema_url":"https://schema.oasf.outshift.com","schema_version":"0.8.0"}' \
  localhost:31234 \
  agntcy.oasfsdk.schema.v1.SchemaService/GetSchemaSkills

grpcurl -plaintext \
  -d '{"schema_url":"https://schema.oasf.outshift.com","schema_version":"0.8.0"}' \
  localhost:31234 \
  agntcy.oasfsdk.schema.v1.SchemaService/GetSchemaDomains

grpcurl -plaintext \
  -d '{"schema_url":"https://schema.oasf.outshift.com","schema_version":"0.8.0"}' \
  localhost:31234 \
  agntcy.oasfsdk.schema.v1.SchemaService/GetSchemaModules
```

## Golang gRPC client example

```go
package main

import (
	"context"
	"log"
	"time"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:31234", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := schemav1grpc.NewSchemaServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	defaultResp, err := client.GetDefaultSchemaVersion(ctx, &schemav1.GetDefaultSchemaVersionRequest{
		SchemaUrl: "https://schema.oasf.outshift.com",
	})
	if err != nil {
		log.Fatalf("GetDefaultSchemaVersion failed: %v", err)
	}
	log.Printf("Default schema version: %s", defaultResp.GetVersion())

	skillsResp, err := client.GetSchemaSkills(ctx, &schemav1.GetSchemaSkillsRequest{
		SchemaUrl:     "https://schema.oasf.outshift.com",
		SchemaVersion: "0.8.0",
	})
	if err != nil {
		log.Fatalf("GetSchemaSkills failed: %v", err)
	}
	log.Printf("Top-level skill categories: %d", len(skillsResp.GetItems()))
}
```

# Validation Service

The OASF SDK Validation Service validates OASF Records using the API validator of the specified OASF schema server via a schema URL.
The validation is performed by the OASF schema server using its own validation logic.

## gRPC API example

**Using schema URL (required):**

```bash
cat tests/fixtures/valid_0.8.0_record.json | jq '{record: ., schema_url: "https://schema.oasf.outshift.com"}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.validation.v1.ValidationService/ValidateRecord
```

## Golang example

```bash
go get github.com/agntcy/oasf-sdk/pkg@v0.0.9
go get buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go@v1.36.10-20251029125108-823ea6fabc82.1
go get buf.build/gen/go/agntcy/oasf-sdk/grpc/go@v1.5.1-20251029125108-823ea6fabc82.2
```

Package based usage:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/agntcy/oasf-sdk/pkg/validator"
)

func main() {
	// Create a new validator instance with schema URL
	v, err := validator.New("https://schema.oasf.outshift.com")
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Sample OASF record data as a Go struct
	recordData := map[string]interface{}{
		"name":           "example.org/my-agent",
		"schema_version": "0.8.0",
		"version":        "v1.0.0",
		"description":    "An example agent for demonstration",
		"authors":        []string{"Your Name <your.email@example.com>"},
		"created_at":     "2025-01-01T00:00:00Z",
		"domains": []map[string]interface{}{
			{
				"id":   101,
				"name": "technology/internet_of_things",
			},
		},
		"locators": []map[string]interface{}{
			{
				"type": "docker_image",
				"url":  "ghcr.io/example/my-agent:latest",
			},
		},
		"skills": []map[string]interface{}{
			{
				"name": "natural_language_processing/natural_language_understanding",
				"id":   101,
			},
		},
	}

	// Convert Go struct to protobuf format using OASF SDK decoder
	recordStruct, err := decoder.StructToProto(recordData)
	if err != nil {
		log.Fatalf("Failed to convert struct to proto: %v", err)
	}

	// Validate the record
	ctx := context.Background()
	isValid, errors, warnings, err := v.ValidateRecord(ctx, recordStruct)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("Record is valid: %t\n", isValid)
	if len(errors) > 0 {
		fmt.Println("Validation errors:")
		for _, errMsg := range errors {
			fmt.Printf("  - %s\n", errMsg)
		}
	} else {
		fmt.Println("No validation errors found!")
	}
	if len(warnings) > 0 {
		fmt.Println("Validation warnings:")
		for _, warnMsg := range warnings {
			fmt.Printf("  - %s\n", warnMsg)
		}
	}
}
```

Service based usage:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/validation/v1/validationv1grpc"
	"buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/validation/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the validation service
	conn, err := grpc.NewClient("localhost:31234", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := validationv1grpc.NewValidationServiceClient(conn)

	authors := []string{"Your Name <your.email@example.com>"}
	authorsIface := make([]interface{}, len(authors))
	for i, v := range authors {
		authorsIface[i] = v
	}

	// Sample OASF record to validate
	record := map[string]interface{}{
		"name":           "example.org/my-agent",
		"schema_version": "0.8.0",
		"version":        "v1.0.0",
		"description":    "An example agent for demonstration",
		"authors":        authorsIface,
		"created_at":     "2025-01-01T00:00:00Z",
		"domains": []map[string]interface{}{
			{
				"id":   101,
				"name": "technology/internet_of_things",
			},
		},
		"locators": []map[string]interface{}{
			{
				"type": "docker_image",
				"url":  "ghcr.io/example/my-agent:latest",
			},
		},
		"skills": []map[string]interface{}{
			{
				"name": "natural_language_processing/natural_language_understanding",
				"id":   101,
			},
		},
	}

	// Convert record to protobuf Struct
	recordProto, err := decoder.StructToProto(record)
	if err != nil {
		log.Fatalf("Failed to convert record to proto: %v", err)
	}

	// Create validation request
	req := &validationv1.ValidateRecordRequest{
		Record:    recordProto,
		// Schema URL is required
		SchemaUrl: "https://schema.oasf.outshift.com",
	}

	// Call validation service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ValidateRecord(ctx, req)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Print results
	fmt.Printf("Valid: %t\n", resp.IsValid)
	if len(resp.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, err := range resp.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
	if len(resp.Warnings) > 0 {
		fmt.Printf("Warnings:\n")
		for _, warn := range resp.Warnings {
			fmt.Printf("  - %s\n", warn)
		}
	}
}
```

## Python example

```bash
uv add 'agntcy-oasf-sdk-grpc-python==1.75.0.1.20250917120021+8b2bf93bf8dc' --index https://buf.build/gen/python
uv add 'agntcy-oasf-sdk-protocolbuffers-python==32.1.0.1.20250917120021+8b2bf93bf8dc' --index https://buf.build/gen/python
```

```python
import grpc
from google.protobuf.struct_pb2 import Struct

from agntcy.oasfsdk.validation.v1.validation_service_pb2 import ValidateRecordRequest
from agntcy.oasfsdk.validation.v1.validation_service_pb2_grpc import ValidationServiceStub

def validate_record():
    # Sample OASF record to validate
    record_data = {
        "name": "example.org/my-agent",
        "schema_version": "0.8.0",
        "version": "v1.0.0",
        "description": "An example agent for demonstration",
        "authors": ["Your Name <your.email@example.com>"],
        "created_at": "2025-01-01T00:00:00Z",
        "domains": [
            {
                "id": 101,
                "name": "technology/internet_of_things"
            }
        ],
        "locators": [
            {
                "type": "docker_image",
                "url": "ghcr.io/example/my-agent:latest"
            }
        ],
        "skills": [
            {
                "name": "natural_language_processing/natural_language_understanding",
                "id": 101
            }
        ]
    }

    # Create gRPC channel
    with grpc.insecure_channel('localhost:31234') as channel:
        stub = ValidationServiceStub(channel)

        # Convert dict to protobuf Struct
        record_struct = Struct()
        record_struct.update(record_data)

        # Create validation request
        request = ValidateRecordRequest(
            record=record_struct,
            schema_url="https://schema.oasf.outshift.com"  # Required
        )

        try:
            # Call validation service
            response = stub.ValidateRecord(request)

            # Print results
            print(f"Valid: {response.is_valid}")
            if response.errors:
                print("Errors:")
                for error in response.errors:
                    print(f"  - {error}")
            else:
                print("No validation errors found!")
            if response.warnings:
                print("Warnings:")
                for warning in response.warnings:
                    print(f"  - {warning}")

        except grpc.RpcError as e:
            print(f"gRPC error: {e.code()}: {e.details()}")

if __name__ == "__main__":
    validate_record()
```

## JavaScript example

```bash
npm config set @buf:registry https://buf.build/gen/npm/v1/
npm install @buf/agntcy_oasf-sdk.grpc_node@1.13.0-20250917120021-8b2bf93bf8dc.2
```

```javascript
const grpc = require("@grpc/grpc-js");
const {
  ValidationServiceClient,
} = require("@buf/agntcy_oasf-sdk.grpc_node/agntcy/oasfsdk/validation/v1/validation_service_grpc_pb");
const {
  ValidateRecordRequest,
} = require("@buf/agntcy_oasf-sdk.grpc_node/agntcy/oasfsdk/validation/v1/validation_service_pb");
const { Struct } = require("google-protobuf/google/protobuf/struct_pb");

async function validateRecord() {
  // Sample OASF record to validate
  const recordData = {
    name: "example.org/my-agent",
    schema_version: "0.8.0",
    version: "v1.0.0",
    description: "An example agent for demonstration",
    authors: ["Your Name <your.email@example.com>"],
    created_at: "2025-01-01T00:00:00Z",
    previous_record_cid: "2883dcaa-ae90-11f0-9e37-5e1f5302e045",
    domains: [
      {
        id: 101,
        name: "technology/internet_of_things",
      },
    ],
    locators: [
      {
        type: "docker_image",
        url: "ghcr.io/example/my-agent:latest",
      },
    ],
    skills: [
      {
        name: "natural_language_processing/natural_language_understanding",
        id: 101,
      },
    ],
  };

  // Create gRPC client
  const client = new ValidationServiceClient(
    "localhost:31234",
    grpc.credentials.createInsecure(),
  );

  // Convert JavaScript object to protobuf Struct using the proper method
  const recordStruct = Struct.fromJavaScript(recordData);

  // Create validation request
  const request = new ValidateRecordRequest();
  request.setRecord(recordStruct);
  // Schema URL is required
  request.setSchemaUrl("https://schema.oasf.outshift.com");

  return new Promise((resolve, reject) => {
    client.validateRecord(request, (error, response) => {
      if (error) {
        console.error("gRPC error:", error);
        reject(error);
        return;
      }

      // Print results
      console.log(`Valid: ${response.getIsValid()}`);
      const errors = response.getErrorsList();
      if (errors && errors.length > 0) {
        console.log("Errors:");
        errors.forEach((err) => {
          console.log(`  - ${err}`);
        });
      } else {
        console.log("No validation errors found!");
      }
      const warnings = response.getWarningsList();
      if (warnings && warnings.length > 0) {
        console.log("Warnings:");
        warnings.forEach((warn) => {
          console.log(`  - ${warn}`);
        });
      }

      resolve(response);
    });
  });
}

// Run the validation
validateRecord()
  .then(() => {
    console.log("Validation completed successfully");
    process.exit(0);
  })
  .catch((error) => {
    console.error("Validation failed:", error);
    process.exit(1);
  });
```

# Extractor

Maps free-form text (a search query or a whole `SKILL.md`) onto the OASF
taxonomy — ranked **skills** and **domains**, literally-mentioned **modules**
(`mcp`, `a2a`, `agentskills`), and a few free-text **keywords**. Semantic ranking
uses `all-MiniLM-L6-v2` run in-process in pure Go (cybertron/spago) — no LLM, no
external inference service. The taxonomy is fetched from a configured OASF
endpoint (via `pkg/schema`); neither the taxonomy nor the model is embedded in
the binary.

There are two ways to use it:

- **In-process (`pkg/extractor`)** — for local tools, `dirctl`, and the importer.
  Provision assets once to a local directory, then load and query in the same
  process. The model is not compiled into your binary (it lives on disk), and a
  warm load from disk is fast (~150 ms), so a one-shot CLI or a batch import runs
  fine without any separate service.
- **Over gRPC (the oasf-sdk server)** — for the hosted case, e.g. a Directory
  node's AI Catalog UI on Kubernetes. Run the server; it provisions its assets at
  startup and serves `ExtractorService.Extract`. Clients call it remotely and
  never compile in the model.

## In-process (`pkg/extractor`)

```go
import "github.com/agntcy/oasf-sdk/pkg/extractor"

// 1. Provision once — downloads+converts the model from HuggingFace and computes
//    label vectors from the taxonomy fetched at oasfURL, writing both to the
//    asset dir (default ~/.agntcy/oasf-sdk/extractor/). Idempotent; re-embeds
//    only when the model or taxonomy changed.
err := extractor.Provision(ctx, extractor.WithOASFURL(oasfURL))

// 2. New — loads the warm engine from the asset dir. No network I/O. Reuse it
//    (safe for concurrent use). WithOASFURL is required; New errors if the
//    assets have not been provisioned or the URL differs from the provisioned one.
e, err := extractor.New(extractor.WithOASFURL(oasfURL))

// 3. Extract — pure and fast.
res, err := e.Extract(ctx, "an agent that reviews code for bugs and speaks mcp")
// res.Skills / res.Domains (ranked, tiered) · res.Modules · res.Keywords
```

Key options: `WithOASFURL` (**required**), `WithModelName` (default
`all-MiniLM-L6-v2`; any cybertron-convertible BERT model), `WithAssetDir`
(default `~/.agntcy/oasf-sdk/extractor/`). Query scope: `All()` (default — every
version the endpoint serves; use for search) or `Latest()` / `Versions(...)`
(use when enriching a record on import). `Tiers(n)` returns the closest `n`
score groups.

`New` loads everything from the asset dir and makes no network calls, so it does
not notice an OASF instance that changed in place. Re-run `Provision`
(idempotent) to refresh after the endpoint's taxonomy changes.

## Hosted (oasf-sdk server)

The server registers the extractor controller whenever an OASF URL is
configured, provisions its assets at startup, then serves the gRPC
`ExtractorService`:

```bash
OASF_SDK_EXTRACTOR_OASF_URL=https://schema.oasf.outshift.com \
go run ./server/cmd            # or the published container image
```

- `OASF_SDK_EXTRACTOR_OASF_URL` — OASF schema endpoint; setting it enables the
  extractor controller (its on/off switch).
- `OASF_SDK_EXTRACTOR_MODEL_NAME` — embedding model (optional; default
  `all-MiniLM-L6-v2`, any cybertron-convertible BERT model).
- `OASF_SDK_EXTRACTOR_ASSET_DIR` — asset directory (optional; defaults as above).
- Optional scoring overrides (each falls back to the library default): 
  `OASF_SDK_EXTRACTOR_SKILL_SEMANTIC_WEIGHT`, `..._SKILL_LEXICAL_WEIGHT`,
  `..._DOMAIN_SEMANTIC_WEIGHT`, `..._DOMAIN_LEXICAL_WEIGHT`, `..._TIERS`,
  `..._TIER_RATIO`, `..._MIN_SCORE`.

On startup it runs `Provision` (idempotent) and loads the engine, so it is
self-sufficient — no separate provisioning step. For Kubernetes, bake the model
into the image at build time to avoid a runtime HuggingFace download and get
fast, reproducible pod starts, and gate the readiness probe on startup
completing. The service is stateless and can run multiple replicas behind a
`ClusterIP` Service.

### Calling it with grpcurl

The server registers gRPC reflection, so [`grpcurl`](https://github.com/fullstorydev/grpcurl)
needs no proto files. Assuming it listens on `0.0.0.0:31234`:

```bash
# discover the service and its request shape
grpcurl -plaintext 0.0.0.0:31234 list
grpcurl -plaintext 0.0.0.0:31234 describe agntcy.oasfsdk.extractor.v1.ExtractorService

# extract from a free-text query (scope + tiers optional)
grpcurl -plaintext -d '{
  "text": "an agent that reviews code for bugs and speaks mcp",
  "scope": "VERSION_SCOPE_ALL",
  "tiers": 1
}' 0.0.0.0:31234 agntcy.oasfsdk.extractor.v1.ExtractorService/Extract

# scope to the newest version (as when enriching a record on import)
grpcurl -plaintext -d '{"text": "summarize legal documents", "scope": "VERSION_SCOPE_LATEST"}' \
  0.0.0.0:31234 agntcy.oasfsdk.extractor.v1.ExtractorService/Extract
```

Request fields: `text`, `scope` (`VERSION_SCOPE_ALL` | `VERSION_SCOPE_LATEST`),
`versions` (explicit pins, overrides `scope`), `tiers`, `minScore`,
`minResults`. The response contains `skills`, `domains`, `modules` (each a
`ScoredClass` with `kind`, `versions`, `score`, `tier`, …) and `keywords`.

### Running the container against a local OASF (dev)

Running the server image on a workstation — especially on a network that
inspects or blocks outbound HTTPS — against a local (e.g. kind) OASF needs two
knobs:

- **Reuse host-provisioned assets** so the container does not fetch the model
  from HuggingFace at runtime. Provision the asset dir on the host first (the
  in-process `Provision` above populates `~/.agntcy/oasf-sdk/extractor`), then
  mount it and set `OASF_SDK_EXTRACTOR_ASSET_DIR` to the mount. The container
  then only fetches the taxonomy from the OASF endpoint (no HuggingFace call).
- **Reach the host's OASF.** Containers reach host services via
  `host.docker.internal`. If that OASF is behind an ingress that virtual-hosts
  on `localhost`, use `localhost:8080` as the URL (so the `Host` header matches
  the ingress) and map `localhost` to the host with `--add-host
  localhost:host-gateway`.

```bash
docker run -p 31234:31234 \
  --add-host localhost:host-gateway \
  -e OASF_SDK_EXTRACTOR_OASF_URL=localhost:8080 \
  -e OASF_SDK_EXTRACTOR_ASSET_DIR=/assets \
  -v "$HOME/.agntcy/oasf-sdk/extractor:/assets" \
  oasf-sdk:latest
```

Against the public endpoint none of this is needed (direct egress, public
certificate, model downloaded at startup) — just set
`OASF_SDK_EXTRACTOR_OASF_URL=https://schema.oasf.outshift.com`.
