# Translation Service

## Prerequisites

- Translation SDK binary, distributed via [GitHub Releases](https://github.com/agntcy/oasf-sdk/releases)
- Translation SDK docker images, distributed via
  [GitHub Packages](https://github.com/orgs/agntcy/packages?repo_name=oasf-sdk)

Let's start the OASF SDK as a docker container, which will listen for incoming requests on port `31234`:

```bash
docker run -p 31234:31234 ghcr.io/agntcy/oasf-sdk:latest
```

## GitHub Copilot config

Create a GitHub Copilot config from the OASF data model using the `RecordToGHCopilot` RPC method.
You can pipe the output to a file wherever you want to save the config.

```bash
cat e2e/fixtures/translation_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToGHCopilot
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
cat e2e/fixtures/translation_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.translation.v1.TranslationService/RecordToA2A
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

# Validation Service

The OASF SDK Validation Service validates OASF Records against [JSON Schema v0.7](https://json-schema.org/draft-07).
It supports two validation modes:
- **Embedded schemas** - Uses JSON schemas built into the binary (default)
- **Schema URL** - Fetches and validates against the schema URL from the record

## gRPC API example

```bash
cat e2e/fixtures/valid_0.7.0_record.json | jq '{record: .}' | grpcurl -plaintext -d @ localhost:31234 agntcy.oasfsdk.validation.v1.ValidationService/ValidateRecord
```

## Golang example

```bash
go get github.com/agntcy/oasf-sdk/pkg@v0.0.5
go get buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go@v1.36.9-20250917120021-8b2bf93bf8dc.1
go get buf.build/gen/go/agntcy/oasf-sdk/grpc/go@v1.5.1-20250917120021-8b2bf93bf8dc.2
```

Package based usage:
```go
package main

import (
	"fmt"
	"log"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/agntcy/oasf-sdk/pkg/validator"
)

func main() {
	// Create a new validator instance with embedded schemas
	v, err := validator.New()
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Sample OASF record data as a Go struct
	recordData := map[string]interface{}{
		"name":           "example.org/my-agent",
		"schema_version": "0.7.0",
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

	// Validate using embedded schemas (default behavior)
	isValid, errors, err := v.ValidateRecord(recordStruct)
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

	// Optional: Validate against a specific schema URL
	isValidURL, errorsURL, err := v.ValidateRecord(
		recordStruct,
		validator.WithSchemaURL("https://schema.oasf.outshift.com"),
	)
	if err != nil {
		log.Fatalf("URL validation failed: %v", err)
	}

	fmt.Printf("Record is valid (URL schema): %t\n", isValidURL)
	if len(errorsURL) > 0 {
		fmt.Println("URL validation errors:")
		for _, errMsg := range errorsURL {
			fmt.Printf("  - %s\n", errMsg)
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
		"schema_version": "0.7.0",
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
		// Optional: specify schema URL to validate against
		// SchemaUrl: "https://schema.oasf.outshift.com",
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
        "schema_version": "0.7.0",
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
            record=record_struct
            # schema_url="https://schema.oasf.outshift.com"  # Optional
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
        schema_version: "0.7.0",
        version: "v1.0.0",
        description: "An example agent for demonstration",
        authors: ["Your Name <your.email@example.com>"],
        created_at: "2025-01-01T00:00:00Z",
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
    // Optional: request.setSchemaUrl("https://schema.oasf.outshift.com");

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
