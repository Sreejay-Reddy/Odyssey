module github.com/sreejay-reddy/odyssey/odyssey-go

go 1.25.5

require (
	capnproto.org/go/capnp/v3 v3.1.0-alpha.2
	github.com/jackc/pgx/v5 v5.10.0
	github.com/spf13/cobra v1.10.2
	github.com/sreejay-reddy/odyssey/protocol v0.0.0-20260924182316-127409d79127
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/colega/zeropool v0.0.0-20230505084239-6fb4a4f75381 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace github.com/sreejay-reddy/odyssey/protocol => ../protocol
