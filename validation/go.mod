module github.com/agntcy/oasf-sdk/validation

go 1.24.4

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.5.1-20250908111150-ac220391cdfd.2
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.8-20250908111150-ac220391cdfd.1
	github.com/agntcy/oasf-sdk/core v0.0.0-00010101000000-000000000000
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
	github.com/xeipuuv/gojsonschema v1.2.0
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.8
)

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.8-00000000000000-bda8dba52bd5.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/sagikazarmark/locafero v0.8.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/agntcy/oasf-sdk/core => ../core
