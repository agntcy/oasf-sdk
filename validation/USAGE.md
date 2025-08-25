# OASF Validation Service

The OASF Validation Service validates OASF Records against JSON Schema v0.7. It supports two validation modes:
- **Embedded schemas** - Uses JSON schemas built into the binary (default)
- **Schema URL** - Fetches and validates against the schema URL from the record

## Environment Variables

- `VALIDATION_SERVER_LISTEN_ADDRESS`: Server listen address (default: `0.0.0.0:31235`)

## 1. As a Go Library

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

## 2. As a gRPC Server

Run the validation service as a standalone server:

```bash
# Simple - just run it (schemas are embedded)
docker run -p 31235:31235 ghcr.io/agntcy/oasf-sdk-validation:latest
```

Then call it from any language that supports gRPC:

### CLI Example (grpcurl)

You can test the validation service from the command line using [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
cat agent.json | grpcurl -plaintext -d @ localhost:31235 validation.v1.ValidationService/ValidateRecord | jq
```

### Python Example

#### Single Record Validation

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

#### Streaming Validation

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

### JavaScript Example

#### Single Record Validation

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

#### Streaming Validation

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
