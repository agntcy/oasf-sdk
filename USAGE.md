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
cat e2e/fixtures/translation_0.8.0_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToGHCopilot
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
cat e2e/fixtures/translation_0.8.0_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToA2A
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

## MCP Registry to OASF Record

To convert an MCP Registry server.json to an OASF record, use the `MCPToRecord` RPC method. This translates the deployment metadata from an MCP server.json file into an OASF 0.8.0 record with the MCP module populated.

**Note:** The MCP Registry server.json contains deployment metadata (packages, remotes, etc.), not runtime capabilities (tools, resources, prompts). Those are discovered via the MCP protocol when a client connects to the server.

```bash
cat e2e/fixtures/translation_mcp.json | jq '{data: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/MCPToRecord
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
	"encoding/json"
	"fmt"
	"log"

	"github.com/agntcy/oasf-sdk/pkg/schema"
)

func main() {
	// Create a new schema instance with schema URL
	s, err := schema.New("https://schema.oasf.outshift.com")
	if err != nil {
		log.Fatalf("Failed to create schema instance: %v", err)
	}

	ctx := context.Background()

	// Get available schema versions from the server
	versions, err := s.GetAvailableSchemaVersions(ctx)
	if err != nil {
		log.Fatalf("Failed to get available schema versions: %v", err)
	}
	fmt.Printf("Available schema versions: %v\n", versions)

	// Get the default schema version (cached after first fetch)
	defaultVersion, err := s.GetDefaultVersion(ctx)
	if err != nil {
		log.Fatalf("Failed to get default version: %v", err)
	}
	fmt.Printf("Default schema version: %s\n", defaultVersion)

	// Get full schema content for version 0.8.0 (using WithVersion option)
	schemaContent, err := s.GetRecordSchemaContent(ctx, schema.WithVersion("0.8.0"))
	if err != nil {
		log.Fatalf("Failed to get schema content: %v", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaContent, &schemaMap); err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}
	fmt.Printf("Schema version 0.8.0 loaded successfully\n")

	// Get skills from schema (using default version - no option needed)
	skillsData, err := s.GetSchemaSkills(ctx)
	if err != nil {
		log.Fatalf("Failed to get skills: %v", err)
	}

	var skillsMap map[string]interface{}
	if err := json.Unmarshal(skillsData, &skillsMap); err != nil {
		log.Fatalf("Failed to parse skills: %v", err)
	}
	fmt.Printf("Found %d skills in schema\n", len(skillsMap))

	// Get domains from schema (using specific version)
	domainsData, err := s.GetSchemaDomains(ctx, schema.WithVersion("0.8.0"))
	if err != nil {
		log.Fatalf("Failed to get domains: %v", err)
	}

	var domainsMap map[string]interface{}
	if err := json.Unmarshal(domainsData, &domainsMap); err != nil {
		log.Fatalf("Failed to parse domains: %v", err)
	}
	fmt.Printf("Found %d domains in schema\n", len(domainsMap))

	// Get a specific $defs key (e.g., modules) using default version
	modulesData, err := s.GetSchemaKey(ctx, "modules")
	if err != nil {
		log.Fatalf("Failed to get modules: %v", err)
	}

	var modulesMap map[string]interface{}
	if err := json.Unmarshal(modulesData, &modulesMap); err != nil {
		log.Fatalf("Failed to parse modules: %v", err)
	}
	fmt.Printf("Found %d modules in schema\n", len(modulesMap))
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

### GetDefaultVersion
Returns the default schema version from the server. The version is cached after the first fetch:
```go
defaultVersion, err := s.GetDefaultVersion(ctx)
```

### GetRecordSchemaContent
Fetches the complete JSON schema. If no version is provided via `WithVersion()`, the default version from the server is used:
```go
// Using default version
schemaContent, err := s.GetRecordSchemaContent(ctx)

// Using specific version
schemaContent, err := s.GetRecordSchemaContent(ctx, schema.WithVersion("0.8.0"))
```

### GetSchemaKey
Extracts a specific `$defs` category from the schema. If no version is provided, the default version is used:
```go
// Using default version
skillsData, err := s.GetSchemaKey(ctx, "skills")

// Using specific version
skillsData, err := s.GetSchemaKey(ctx, "skills", schema.WithVersion("0.8.0"))
domainsData, err := s.GetSchemaKey(ctx, "domains", schema.WithVersion("0.8.0"))
modulesData, err := s.GetSchemaKey(ctx, "modules", schema.WithVersion("0.8.0"))
```

### Convenience Methods
All convenience methods accept optional `WithVersion()` option. If omitted, the default version is used:
- `GetSchemaSkills(ctx, ...SchemaOption)` - Extracts skills definitions
- `GetSchemaDomains(ctx, ...SchemaOption)` - Extracts domains definitions
- `GetSchemaModules(ctx, ...SchemaOption)` - Extracts modules definitions

Example:
```go
// Using default version
skills, err := s.GetSchemaSkills(ctx)

// Using specific version
skills, err := s.GetSchemaSkills(ctx, schema.WithVersion("0.7.0"))
```

# Validation Service

The OASF SDK Validation Service validates OASF Records using the API validator of the specified OASF schema server via a schema URL.
The validation is performed by the OASF schema server using its own validation logic.

## gRPC API example

**Using schema URL (required):**
```bash
cat e2e/fixtures/valid_0.8.0_record.json | jq '{record: ., schema_url: "https://schema.oasf.outshift.com"}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.validation.v1.ValidationService/ValidateRecord
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
const grpc = require('@grpc/grpc-js');
const { ValidationServiceClient } = require('@buf/agntcy_oasf-sdk.grpc_node/agntcy/oasfsdk/validation/v1/validation_service_grpc_pb');
const { ValidateRecordRequest } = require('@buf/agntcy_oasf-sdk.grpc_node/agntcy/oasfsdk/validation/v1/validation_service_pb');
const { Struct } = require('google-protobuf/google/protobuf/struct_pb');

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
                name: "technology/internet_of_things"
            }
        ],
        locators: [
            {
                type: "docker_image",
                url: "ghcr.io/example/my-agent:latest"
            }
        ],
        skills: [
            {
                name: "natural_language_processing/natural_language_understanding",
                id: 101
            }
        ]
    };

    // Create gRPC client
    const client = new ValidationServiceClient(
        'localhost:31234',
        grpc.credentials.createInsecure()
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
                console.error('gRPC error:', error);
                reject(error);
                return;
            }

            // Print results
            console.log(`Valid: ${response.getIsValid()}`);
            const errors = response.getErrorsList();
            if (errors && errors.length > 0) {
                console.log('Errors:');
                errors.forEach(err => {
                    console.log(`  - ${err}`);
                });
            } else {
                console.log('No validation errors found!');
            }
            const warnings = response.getWarningsList();
            if (warnings && warnings.length > 0) {
                console.log('Warnings:');
                warnings.forEach(warn => {
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
        console.log('Validation completed successfully');
        process.exit(0);
    })
    .catch((error) => {
        console.error('Validation failed:', error);
        process.exit(1);
    });
```
