# testrunner

`testrunner` is a simple test runner for Go. In order to use it just run in in the root dir of a Go project -- it will detect changes to any `*.go` files in the directory subtree and automatically run the `go test -race [package]` command for the package with changed code.

## Features

- automatically detects directory tree changes (added/deleted directories)
- runs only on changes to `.go` files

## TODO

- tests (integration?)
- configuration (verbose mode?, configurable root directory?)
