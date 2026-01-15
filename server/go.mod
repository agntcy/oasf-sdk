module github.com/agntcy/oasf-sdk/server

go 1.24.4

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.6.0-20260115113053-9b110d5996b7.1
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.11-20260115113053-9b110d5996b7.1
	github.com/agntcy/oasf-sdk/pkg v0.0.14
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/spf13/cobra v1.10.1
	github.com/spf13/viper v1.21.0
	google.golang.org/grpc v1.75.1
)

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.11-20251022143645-07a420b66e81.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/agntcy/oasf-sdk/pkg => ../pkg
