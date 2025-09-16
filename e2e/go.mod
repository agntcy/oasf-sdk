module github.com/agntcy/oasf-sdk/e2e

go 1.24.4

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.5.1-20250915083413-97472ce2fb9f.2
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.9-20250915083413-97472ce2fb9f.1
	github.com/agntcy/oasf-sdk/pkg v0.0.4
	github.com/onsi/ginkgo/v2 v2.22.0
	github.com/onsi/gomega v1.36.1
	google.golang.org/grpc v1.74.2
)

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.9-20250915080448-a3d79079b9ac.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/agntcy/oasf-sdk/pkg => ../pkg
