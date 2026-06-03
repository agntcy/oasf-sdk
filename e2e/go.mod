module github.com/agntcy/oasf-sdk/e2e

go 1.26.1

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.6.2-20260603092525-c7b60e1e21a1.1
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.11-20260603092525-c7b60e1e21a1.1
	github.com/agntcy/oasf-sdk/pkg v1.0.5
	github.com/onsi/ginkgo/v2 v2.22.0
	github.com/onsi/gomega v1.36.1
	google.golang.org/grpc v1.81.0
)

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.11-20260409142051-fd433ebe75bb.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/agntcy/oasf-sdk/pkg => ../pkg
