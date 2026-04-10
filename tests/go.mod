module github.com/agntcy/oasf-sdk/tests

go 1.26.1

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.6.1-20260410103700-b5956310ea54.1
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.11-20260410103700-b5956310ea54.1
	github.com/agntcy/oasf-sdk/pkg v1.0.5
	github.com/onsi/ginkgo/v2 v2.22.0
	github.com/onsi/gomega v1.36.1
	google.golang.org/grpc v1.78.0
)

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.11-20260409142051-fd433ebe75bb.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/agntcy/oasf-sdk/pkg => ../pkg
