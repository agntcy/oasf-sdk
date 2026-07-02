module github.com/agntcy/oasf-sdk/server

go 1.26.1

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.6.2-20260702111013-9662f012527d.1
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.11-20260702111013-9662f012527d.1
	github.com/agntcy/oasf-sdk/pkg v1.0.5
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/spf13/cobra v1.10.1
	github.com/spf13/viper v1.21.0
	google.golang.org/grpc v1.81.0
)

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.11-20260409142051-fd433ebe75bb.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/nlpodyssey/cybertron v0.2.1 // indirect
	github.com/nlpodyssey/gopickle v0.2.0 // indirect
	github.com/nlpodyssey/gotokenizers v0.2.0 // indirect
	github.com/nlpodyssey/spago v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/rs/zerolog v1.31.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/agntcy/oasf-sdk/pkg => ../pkg
