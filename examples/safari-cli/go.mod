module github.com/tmc/macgo/examples/safari-cli

go 1.24.0

toolchain go1.24.6

require (
	github.com/spf13/cobra v1.8.1
	github.com/tmc/macgo v0.0.0
)

replace github.com/tmc/macgo => ../..

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.39.0 // indirect
)
