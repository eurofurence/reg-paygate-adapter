# reg-paygate-adapter

<img src="https://github.com/eurofurence/reg-paygate-adapter/actions/workflows/go.yml/badge.svg" alt="test status"/>

## Overview

A backend service that provides payment functionality via the Nexi payment provider.

Implemented in Go.

Command line arguments
```
-config <path-to-config-file> [-ecs-json-logging]
```

## Installation

This service uses go modules to provide dependency management, see `go.mod`.

If you place this repository outside your GOPATH, build and test runs will download all required
dependencies by default.

## Running on localhost

Copy the configuration template from `docs/config-template.yaml` to `./config.yaml`. This will set you up
for operation with an in-memory database and sensible defaults.

Build using `go build cmd/main.go`.

Then run `./main -config config.yaml`.

## Installation on the server

See `install.sh`. This assumes a current build, and a valid configuration template in specific filenames.

## Test Coverage

In order to collect full test coverage, set go tool arguments to `-covermode=atomic -coverpkg=./internal/...`,
or manually run
```
go test -covermode=atomic -coverpkg=./internal/... ./...
```

## Open Issues and Ideas

We track open issues as GitHub issues on this repository once it becomes clear what exactly needs to be done.
