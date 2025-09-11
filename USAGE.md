# OASF SDK

## Translation SDK

### Prerequisites

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

### VSCode MCP Config

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

### A2A Card extraction

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

## Validation SDK

The OASF Validation Service validates OASF Records against JSON Schema v0.7. It supports two validation modes:
- **Embedded schemas** - Uses JSON schemas built into the binary (default)
- **Schema URL** - Fetches and validates against the schema URL from the record

### Environment Variables

- `VALIDATION_SERVER_LISTEN_ADDRESS`: Server listen address (default: `0.0.0.0:31235`)

### 1. As a Go Library

Import the validation service directly into your Go project:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/agntcy/oasf-sdk/validation/service"
    validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/validation/v1"
    objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
)

func main() {
    // Create validation service (schemas are embedded in the binary)
    validator, err := service.NewValidationService()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create a record to validate
    record := &objectsv3.Record{
        Id:      "my-record",
        Name:    "Test Record",
        Version: "0.5.0",
        // ... other fields
    }
    
    // Option 1: Validate against embedded schemas (default)
    req := &validationv1.ValidateRecordRequest{
        Record:    record,
        SchemaUrl: "", // Empty string uses embedded schemas
    }
    
    // Option 2: Validate against a specific schema URL
    req = &validationv1.ValidateRecordRequest{
        Record:    record,
        SchemaUrl: "https://example.com/schemas/v0.5.0.json", // Provide schema URL
    }
    
    // Validate the record
    isValid, errors, err := validator.ValidateRecord(req)
    if err != nil {
        log.Fatal(err)
    }
    
    if isValid {
        fmt.Printf("Record %s is valid!\n", record.Id)
    } else {
        fmt.Printf("Record %s is invalid:\n", record.Id)
        for _, err := range errors {
            fmt.Printf("  - %s\n", err)
        }
    }
}
```

### 2. As a gRPC Server

Run the validation service as a standalone server:

```bash
# Simple - just run it (schemas are embedded)
docker run -p 31235:31235 ghcr.io/agntcy/oasf-sdk-validation:latest
```

Then call it from any language that supports gRPC:

#### CLI Example (grpcurl)

You can test the validation service from the command line using [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
cat agent.json | grpcurl -plaintext -d @ localhost:31235 validation.v1.ValidationService/ValidateRecord | jq
```

#### Python Example

##### Single Record Validation

```python
import grpc
from validation.v1 import validation_service_pb2_grpc, validation_service_pb2

channel = grpc.insecure_channel('localhost:31235')
stub = validation_service_pb2_grpc.ValidationServiceStub(channel)

// Validate against embedded schemas
request = validation_service_pb2.ValidateRecordRequest(
    record=your_record,
    schema_url=""  # Empty string for embedded schemas
)

# Or validate against specific schema URL
request = validation_service_pb2.ValidateRecordRequest(
    record=your_record,
    schema_url="https://example.com/schemas/v0.5.0.json"
)

response = stub.ValidateRecord(request)

if response.is_valid:
    print("Record is valid!")
else:
    print(f"Validation errors: {response.errors}")
```

##### Streaming Validation

```python
def generate_requests():
    for record in your_records:
        yield validation_service_pb2.ValidateRecordStreamRequest(
            record=record,
            schema_url=""  # Empty for embedded, or provide URL string
        )

responses = stub.ValidateRecordStream(generate_requests())
for response in responses:
    if response.is_valid:
        print("Record is valid!")
    else:
        print(f"Validation errors: {response.errors}")
```

#### JavaScript Example

##### Single Record Validation

```javascript
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');

const packageDefinition = protoLoader.loadSync('validation_service.proto');
const validationService = grpc.loadPackageDefinition(packageDefinition).validation.v1;

const client = new validationService.ValidationService('localhost:31235', grpc.credentials.createInsecure());

// Single record validation
client.ValidateRecord({
    record: yourRecord,
    schema_url: ""  // Empty for embedded schemas, or provide URL
}, (error, response) => {
    if (error) {
        console.error(error);
        return;
    }
    
    if (response.isValid) {
        console.log('Record is valid!');
    } else {
        console.log('Validation errors:', response.errors);
    }
});
```

##### Streaming Validation

```javascript
// Streaming validation
const stream = client.ValidateRecordStream();
stream.on('data', (response) => {
    if (response.isValid) {
        console.log('Record is valid!');
    } else {
        console.log('Validation errors:', response.errors);
    }
});

// Send records to stream
yourRecords.forEach(record => {
    stream.write({
        record: record,
        schema_url: ""  // Empty for embedded, or provide URL string
    });
});
stream.end();
```
