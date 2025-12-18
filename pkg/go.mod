module github.com/agntcy/oasf-sdk/pkg

go 1.24.4

require (
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.10-20251029125108-823ea6fabc82.1
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.10-20251022143645-07a420b66e81.1
	github.com/kagenti/operator v0.0.0-00010101000000-000000000000
	github.com/xeipuuv/gojsonschema v1.2.0
	google.golang.org/protobuf v1.36.10
	k8s.io/api v0.32.0
	k8s.io/apimachinery v0.32.0
)

replace github.com/kagenti/operator => github.com/kagenti/kagenti-operator/kagenti-operator v0.0.0-20251209235923-207524f24e65

require (
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738 // indirect
	sigs.k8s.io/controller-runtime v0.20.0 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
