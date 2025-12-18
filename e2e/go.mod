module github.com/agntcy/oasf-sdk/e2e

go 1.24.4

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.5.1-20251104080327-0fc042e98377.2
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.10-20251104080327-0fc042e98377.1
	github.com/agntcy/oasf-sdk/pkg v0.0.14
	github.com/onsi/ginkgo/v2 v2.22.0
	github.com/onsi/gomega v1.36.1
	google.golang.org/grpc v1.74.2
)

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.10-20251022143645-07a420b66e81.1 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kagenti/operator v0.0.0-00010101000000-000000000000 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.32.0 // indirect
	k8s.io/apimachinery v0.32.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738 // indirect
	sigs.k8s.io/controller-runtime v0.20.0 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/agntcy/oasf-sdk/pkg => ../pkg

// Replace directive for github.com/kagenti/operator
//
// This module declares github.com/kagenti/operator but the repository is
// github.com/kagenti/kagenti-operator with go.mod in subdirectory kagenti-operator/kagenti-operator/.
//
// Using a commit hash (instead of version tag) allows Go to resolve the subdirectory path.
// The commit hash corresponds to v0.2.0-alpha.18 tag.
//
// This is only needed if you want to run e2e tests that use RecordToKagentiAgentSpec.
replace github.com/kagenti/operator => github.com/kagenti/kagenti-operator/kagenti-operator v0.0.0-20251209235923-207524f24e65
