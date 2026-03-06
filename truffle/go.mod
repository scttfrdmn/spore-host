module github.com/scttfrdmn/spore-host/truffle

go 1.24

require (
	github.com/aws/aws-sdk-go-v2 v1.41.3
	github.com/aws/aws-sdk-go-v2/config v1.32.11
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.294.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.96.0
	github.com/aws/aws-sdk-go-v2/service/servicequotas v1.34.3
	github.com/fatih/color v1.18.0
	github.com/olekukonko/tablewriter v1.1.3
	github.com/scttfrdmn/spore-host/pkg/i18n v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.11 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.8 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/clipperhouse/displaywidth v0.6.2 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/nicksnyder/go-i18n/v2 v2.4.1 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.1.0 // indirect
	github.com/olekukonko/ll v0.1.4-0.20260115111900-9e59c2286df0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

// Package truffle provides AWS EC2 instance type discovery and analysis.
//
// This module can be used both as a CLI tool and as a library in other Go applications.
//
// Key packages:
//   - github.com/scttfrdmn/spore-host/truffle/pkg/aws: AWS EC2 client and data structures
//   - github.com/scttfrdmn/spore-host/truffle/pkg/output: Output formatting utilities
//
// Example usage as a library:
//   import "github.com/scttfrdmn/spore-host/truffle/pkg/aws"
//
//   client, _ := aws.NewClient(ctx)
//   results, _ := client.SearchInstanceTypes(ctx, regions, matcher, opts)

replace github.com/scttfrdmn/spore-host/pkg/i18n => ../pkg/i18n
