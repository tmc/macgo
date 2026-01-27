module app-to-web

go 1.25.5

replace github.com/tmc/appledocs => /Users/tmc/go/src/github.com/tmc/appledocs

require (
	github.com/ebitengine/purego v0.9.1
	github.com/tmc/appledocs/generated v0.0.0-00010101000000-000000000000
	github.com/tmc/macgo v0.0.0
)

replace github.com/tmc/macgo => ../..

replace github.com/tmc/appledocs/generated => /Users/tmc/go/src/github.com/tmc/appledocs/generated
